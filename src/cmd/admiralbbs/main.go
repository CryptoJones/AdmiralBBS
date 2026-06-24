// Command admiralbbs is the BBS daemon: it serves the Telnet and SSH
// transports, wraps each caller in a hardened, encrypted session, and routes
// them through the menu engine. See docs/ARCHITECTURE.md.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"admiralbbs/src/audit"
	"admiralbbs/src/crypto"
	"admiralbbs/src/doors"
	"admiralbbs/src/menu"
	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
	"admiralbbs/src/transport"

	"golang.org/x/crypto/ssh"
)

const sysopLevel = 100

// version is the released BBS version (SemVer). Bump the PATCH on each merge;
// MINOR for backward-compatible features, MAJOR for breaking changes.
const version = "1.1.0"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	telnetAddr := flag.String("telnet", ":2323", "telnet listen address (apply-only)")
	sshAddr := flag.String("ssh", ":2222", "ssh listen address (members)")
	hostKey := flag.String("hostkey", "ssh_host_ed25519_key", "ssh host key path")
	auditPath := flag.String("audit", "audit.jsonl", "encrypted audit log path")
	dbPath := flag.String("db", "admiralbbs.db", "SQLite database path")
	saltPath := flag.String("salt", "key.salt", "KDF salt path (non-secret)")
	artPath := flag.String("art", "", "optional custom welcome .ans (overrides the generated name/tagline banner)")
	maxSessions := flag.Int("max-sessions", 100, "max concurrent callers")
	perIP := flag.Int("per-ip", 5, "max concurrent callers per IP")
	idle := flag.Duration("idle", 10*time.Minute, "idle disconnect timeout")
	dailyMinutes := flag.Int("daily-minutes", 60, "default per-member daily time budget (SysOps unlimited)")
	doorUID := flag.Int("door-uid", 0, "run door games as this uid (0=off; needs root)")
	doorGID := flag.Int("door-gid", 0, "gid paired with -door-uid")
	doorChroot := flag.String("door-chroot", "", "chroot door games into this dir (Linux; needs /bin/sh inside)")
	doorNoNet := flag.Bool("door-no-network", false, "run door games with no network (Linux; needs root)")
	doorIsolate := flag.Bool("door-isolate", false, "run door games in fresh namespaces (Linux; needs root)")
	maxPerUser := flag.Int("max-per-user", 1, "max concurrent sessions per user (one node per caller)")
	nodes := flag.Int("nodes", 64, "max concurrent member sessions (node count)")
	doorsDataFlag := flag.String("doors-data", "", "persistent door data dir (default <db-dir>/doors-data)")
	cowboyAddr := flag.String("cowboy", "", "register 'Console Cowboy 2026' resident door at this addr (e.g. 127.0.0.1:4000)")
	flag.Parse()
	if *showVersion {
		fmt.Println("AdmiralBBS " + version)
		return
	}

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

	// Optional shared SysOp-panel password: a step-up secret prompted before the
	// control panel opens (so an unattended logged-in SysOp session can't be used
	// to change BBS settings). Empty = no extra gate.
	sysopPanelPass := os.Getenv("ADMIRALBBS_SYSOP_PASS")
	os.Unsetenv("ADMIRALBBS_SYSOP_PASS")

	db, err := store.Open(*dbPath, vault)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if err := db.EnsureSeedAreas(); err != nil {
		log.Fatalf("seed message areas: %v", err)
	}
	if err := db.EnsureSeedFileAreas(); err != nil {
		log.Fatalf("seed file areas: %v", err)
	}
	if err := db.EnsureSeedDoors(); err != nil {
		log.Fatalf("seed doors: %v", err)
	}
	if *cowboyAddr != "" {
		if err := db.Doors().EnsureResidentDoor("Console Cowboy 2026", "tcp", *cowboyAddr, 0); err != nil {
			log.Fatalf("register Console Cowboy 2026 door: %v", err)
		}
		log.Printf("Console Cowboy 2026 resident door -> %s", *cowboyAddr)
	}
	log.Printf("AdmiralBBS %s starting", version)
	log.Printf("database ready at %s (WAL, encrypted at rest)", *dbPath)

	// Audit: encrypted + hash-chained JSONL is authoritative; session_log mirrors.
	logger, err := audit.New(*auditPath, vault, db.SessionLog())
	if err != nil {
		log.Fatalf("audit log: %v", err)
	}
	defer logger.Close()

	// Graceful shutdown: flush + close state cleanly on SIGINT/SIGTERM (e.g.
	// container stop), since os.Exit skips deferred closes.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigc
		log.Println("shutting down — flushing audit + closing database")
		logger.Close()
		db.Close()
		vault.Close()
		os.Exit(0)
	}()

	presence := session.NewPresence(*maxPerUser)
	nodePool := session.NewNodePool(*nodes)
	roster := session.NewRoster(nil)
	doorsData := *doorsDataFlag
	if doorsData == "" {
		doorsData = filepath.Join(filepath.Dir(*dbPath), "doors-data")
	}

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
		_ = menu.RunApply(s, db)
	}

	// SSH first factor: the offered key must belong to an approved user with
	// that handle (transport-layer auth). The password is the second factor,
	// prompted by the login flow below.
	authenticator := func(username string, key ssh.PublicKey) bool {
		u, err := db.Users().ByHandle(username)
		if err != nil || u.Status != store.StatusApproved {
			return false
		}
		ok, _ := db.Keys().Authorizes(u.ID, key)
		return ok
	}

	// SysOp IP banlist: both transports drop a banned source at accept time,
	// before auth (store.Bans.IsBanned fails open on DB error / bad input).
	banned := func(ip string) bool { return db.Bans().IsBanned(ip) }

	sshHandle := func(c transport.Conn) {
		s := mkSession(c)
		defer s.Close()
		u, ok := menu.RunLogin(s, db) // second factor (password / onboarding)
		if !ok {
			return
		}
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		// One live session per user, NEWEST WINS: a new login displaces any
		// existing session for this handle (intuitive UX + self-heals stale/ghost
		// sessions). Still prevents budget-multiplying across many sessions.
		presID := presence.Enter(u.Handle, func() {
			_, _ = c.Write([]byte("\r\n*** Logged in from another location — this session ends. NO CARRIER ***\r\n"))
			_ = c.Close()
		})
		defer presence.Leave(u.Handle, presID)
		node := nodePool.Acquire()
		if node == 0 {
			w.ColorLine(screen.Red, "All nodes are busy right now — try again shortly. NO CARRIER")
			return
		}
		defer nodePool.Release(node)

		roster.Join(node, u.Handle, s.IP(), s.Transport())
		defer roster.Leave(node)

		enforceBudget(s, db, u, *dailyMinutes)

		// Message of the day: shown once before the menu; caller must press SPACE.
		if motd := db.Settings().MOTD(); motd != "" {
			if err := menu.ShowMOTD(s, motd); err != nil {
				return
			}
		}

		doorOpts := doors.Opts{RunAsUID: *doorUID, RunAsGID: *doorGID, Chroot: *doorChroot, NoNetwork: *doorNoNet, Isolate: *doorIsolate}
		_ = menu.Member(db, u, *artPath, *auditPath, doorOpts, node, doorsData, roster, sysopPanelPass).Run(s)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Printf("telnet (apply-only) listening on %s", *telnetAddr)
		if err := transport.ServeTelnet(*telnetAddr, limits, banned, telnetHandle); err != nil {
			log.Printf("telnet server stopped: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Printf("ssh listening on %s (host key %s)", *sshAddr, *hostKey)
		if err := transport.ServeSSH(*sshAddr, *hostKey, limits, authenticator, banned, sshHandle); err != nil {
			log.Printf("ssh server stopped: %v", err)
		}
	}()

	wg.Wait()
}

// enforceBudget caps a non-SysOp member's session to their remaining daily
// minutes. SysOps are unlimited.
func enforceBudget(s *session.Session, db *store.Store, u *store.User, defaultMinutes int) {
	if u.AccessLevel >= sysopLevel {
		return
	}
	budget := u.DailyMinutes
	if budget <= 0 {
		budget = defaultMinutes
	}
	used, _ := db.SessionLog().MinutesToday(u.Handle)
	remaining := float64(budget) - used
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	if remaining <= 0 {
		w.ColorLine(screen.Red, "Your daily time is used up. Come back tomorrow! NO CARRIER")
		s.Close()
		return
	}
	w.Printf("You have ~%d minutes left today.\r\n", int(remaining))
	s.WatchBudget(time.Duration(remaining * float64(time.Minute)))
}
