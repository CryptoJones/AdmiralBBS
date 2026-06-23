// Command admiralbbs is the BBS daemon: it serves the Telnet and SSH
// transports, wraps each caller in a hardened session, and routes them through
// the menu engine. See docs/ARCHITECTURE.md.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"

	"admiralbbs/src/audit"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/transport"
)

func main() {
	telnetAddr := flag.String("telnet", ":2323", "telnet listen address")
	sshAddr := flag.String("ssh", ":2222", "ssh listen address")
	hostKey := flag.String("hostkey", "ssh_host_ed25519_key", "ssh host key path")
	auditPath := flag.String("audit", "audit.jsonl", "audit log (JSONL) path")
	artPath := flag.String("art", "art/welcome.ans", "welcome screen .ans path")
	flag.Parse()

	// Hardening posture: never run privileged. Doors and the daemon both run
	// as an unprivileged user (DECISIONS.md).
	if os.Geteuid() == 0 {
		log.Fatal("refusing to run as root — start AdmiralBBS as an unprivileged user")
	}

	logger, err := audit.New(*auditPath)
	if err != nil {
		log.Fatalf("audit log: %v", err)
	}
	defer logger.Close()

	mainMenu := menu.Demo(*artPath)
	var counter atomic.Uint64

	handle := func(c transport.Conn) {
		id := fmt.Sprintf("s-%06d", counter.Add(1))
		s := session.New(id, c, logger, nil)
		defer s.Close()
		_ = mainMenu.Run(s)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Printf("telnet listening on %s", *telnetAddr)
		if err := transport.ServeTelnet(*telnetAddr, handle); err != nil {
			log.Printf("telnet server stopped: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Printf("ssh listening on %s (host key %s)", *sshAddr, *hostKey)
		if err := transport.ServeSSH(*sshAddr, *hostKey, handle); err != nil {
			log.Printf("ssh server stopped: %v", err)
		}
	}()

	wg.Wait()
}
