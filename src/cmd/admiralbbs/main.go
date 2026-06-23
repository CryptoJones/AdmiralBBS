// Command admiralbbs is the BBS daemon: it serves the Telnet and SSH
// transports, wraps each caller in a hardened, encrypted session, and routes
// them through the menu engine. See docs/ARCHITECTURE.md.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"admiralbbs/src/audit"
	"admiralbbs/src/crypto"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
	"admiralbbs/src/transport"
)

func main() {
	telnetAddr := flag.String("telnet", ":2323", "telnet listen address (apply-only)")
	sshAddr := flag.String("ssh", ":2222", "ssh listen address (members)")
	hostKey := flag.String("hostkey", "ssh_host_ed25519_key", "ssh host key path")
	auditPath := flag.String("audit", "audit.jsonl", "encrypted audit log path")
	dbPath := flag.String("db", "admiralbbs.db", "SQLite database path")
	saltPath := flag.String("salt", "key.salt", "KDF salt path (non-secret)")
	artPath := flag.String("art", "art/welcome.ans", "welcome screen .ans path")
	maxSessions := flag.Int("max-sessions", 100, "max concurrent callers")
	perIP := flag.Int("per-ip", 5, "max concurrent callers per IP")
	idle := flag.Duration("idle", 10*time.Minute, "idle disconnect timeout")
	flag.Parse()

	// Hardening posture: never run privileged (DECISIONS.md).
	if os.Geteuid() == 0 {
		log.Fatal("refusing to run as root — start AdmiralBBS as an unprivileged user")
	}

	// Encryption is mandatory. The key never touches the data volume or chat.
	secret := os.Getenv("ADMIRALBBS_KEY")
	if secret == "" {
		log.Fatal("ADMIRALBBS_KEY is required — encryption is mandatory (set it via env / Docker secret)")
	}
	salt, err := crypto.LoadOrCreateSalt(*saltPath)
	if err != nil {
		log.Fatalf("salt: %v", err)
	}
	vault, err := crypto.NewVault([]byte(secret), salt)
	if err != nil {
		log.Fatalf("vault: %v", err)
	}
	defer vault.Close()
	os.Unsetenv("ADMIRALBBS_KEY") // shrink the window the secret sits in env

	db, err := store.Open(*dbPath, vault)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	log.Printf("database ready at %s (WAL, encrypted at rest)", *dbPath)

	// Audit: encrypted + hash-chained JSONL is authoritative; session_log mirrors.
	logger, err := audit.New(*auditPath, vault, db.SessionLog())
	if err != nil {
		log.Fatalf("audit log: %v", err)
	}
	defer logger.Close()

	mainMenu := menu.Demo(*artPath)
	var counter atomic.Uint64
	limits := transport.Limits{MaxSessions: *maxSessions, PerIP: *perIP, HandshakeTimeout: 10 * time.Second}

	mkSession := func(c transport.Conn) *session.Session {
		id := fmt.Sprintf("s-%06d", counter.Add(1))
		s := session.New(id, c, logger, nil)
		s.WatchIdle(*idle)
		return s
	}

	// Telnet is apply-only: a caller can reach the membership application and
	// nothing else (DECISIONS: SSH for everything after that).
	telnetHandle := func(c transport.Conn) {
		s := mkSession(c)
		defer s.Close()
		_ = menu.RunApply(s, db.Users(), db.Memberships(), db.Keys())
	}

	// SSH is the members' entrance (full BBS). Login / 2FA enforcement is S003;
	// for now it lands on the main menu.
	sshHandle := func(c transport.Conn) {
		s := mkSession(c)
		defer s.Close()
		_ = mainMenu.Run(s)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Printf("telnet (apply-only) listening on %s", *telnetAddr)
		if err := transport.ServeTelnet(*telnetAddr, limits, telnetHandle); err != nil {
			log.Printf("telnet server stopped: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Printf("ssh listening on %s (host key %s)", *sshAddr, *hostKey)
		if err := transport.ServeSSH(*sshAddr, *hostKey, limits, sshHandle); err != nil {
			log.Printf("ssh server stopped: %v", err)
		}
	}()

	wg.Wait()
}
