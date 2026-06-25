// Command sysopctl is the operator's bootstrap/management tool, run on the BBS
// host (it needs ADMIRALBBS_KEY). It exists primarily to create the FIRST SysOp
// — without it, the access-gated control panel is unreachable on a fresh BBS.
//
//	ADMIRALBBS_KEY=... sysopctl -db data/admiralbbs.db -salt data/key.salt list
//	ADMIRALBBS_KEY=... sysopctl approve <handle> [level]   # approve + print token
//	ADMIRALBBS_KEY=... sysopctl promote <handle> [level]   # set status+level directly
//
// bootstrap creates a ready-to-use SysOp in ONE step — handle + SSH public key +
// password — so an operator never has to do the telnet-apply → onboard dance to
// stand up the first account. It's the self-service path for admins without an
// agent. The password comes from $SYSOP_PASSWORD (not echoed) or, if unset, is
// read from stdin:
//
//	ADMIRALBBS_KEY=... SYSOP_PASSWORD=... \
//	  sysopctl -db data/admiralbbs.db -salt data/key.salt \
//	  bootstrap <handle> <pubkey-file|-> [level]
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"admiralbbs/src/crypto"
	"admiralbbs/src/store"

	"golang.org/x/crypto/ssh"
)

func main() {
	dbPath := flag.String("db", "admiralbbs.db", "SQLite database path")
	saltPath := flag.String("salt", "key.salt", "KDF salt path")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("usage: sysopctl [-db ...] [-salt ...] list | approve <handle> [level=1; 100=SysOp] | promote <handle> [level=1; 100=SysOp] | bootstrap <handle> <pubkey-file|-> [level] | addkey <handle> <pubkey-file|->")
	}

	secret := os.Getenv("ADMIRALBBS_KEY")
	if secret == "" {
		log.Fatal("set ADMIRALBBS_KEY in the env")
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
	db, err := store.Open(*dbPath, vault)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	switch args[0] {
	case "list":
		users, err := db.Users().All()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%-18s %-10s %-6s %s\n", "HANDLE", "STATUS", "LEVEL", "PASSWORD")
		for _, u := range users {
			pw := "set"
			if u.PasswordHash == "" {
				pw = "(unset)"
			}
			fmt.Printf("%-18s %-10s %-6d %s\n", u.Handle, u.Status, u.AccessLevel, pw)
		}

	case "approve", "promote":
		if len(args) < 2 {
			log.Fatalf("usage: sysopctl %s <handle> [level]", args[0])
		}
		handle := args[1]
		// Default to a regular member; SysOp (100) must be granted EXPLICITLY so a
		// bare `approve <handle>` can never silently hand a stranger full admin.
		level := 1
		if len(args) >= 3 {
			if v, e := strconv.Atoi(args[2]); e == nil {
				level = v
			}
		}
		u, err := db.Users().ByHandle(handle)
		if err != nil {
			log.Fatalf("no such user %q: %v", handle, err)
		}
		if err := db.Users().Approve(u.ID, level); err != nil {
			log.Fatalf("approve: %v", err)
		}
		fmt.Printf("%s is now approved at access level %d.\n", handle, level)
		if args[0] == "approve" || u.PasswordHash == "" {
			tok, err := db.Tokens().Issue(u.ID)
			if err != nil {
				log.Fatalf("issue token: %v", err)
			}
			fmt.Printf("One-time onboarding token (relay out-of-band; used once on first SSH login):\n  %s\n", tok)
		}

	case "bootstrap":
		// bootstrap <handle> <pubkey-file|-> [level]
		if len(args) < 3 {
			log.Fatalf("usage: sysopctl bootstrap <handle> <pubkey-file|-> [level]")
		}
		handle := args[1]
		level := 100
		if len(args) >= 4 {
			if v, e := strconv.Atoi(args[3]); e == nil {
				level = v
			}
		}

		// Public key: from a file, or "-" for stdin.
		var keyBytes []byte
		if args[2] == "-" {
			keyBytes, err = io.ReadAll(os.Stdin)
		} else {
			keyBytes, err = os.ReadFile(args[2])
		}
		if err != nil {
			log.Fatalf("read public key: %v", err)
		}
		pubLine := strings.TrimSpace(string(keyBytes))
		if err := store.ValidatePublicKey(pubLine); err != nil {
			log.Fatalf("invalid SSH public key: %v", err)
		}

		// Password: $SYSOP_PASSWORD (not echoed) or a line from stdin.
		pw := os.Getenv("SYSOP_PASSWORD")
		if pw == "" {
			fmt.Fprint(os.Stderr, "SysOp password (input is visible): ")
			line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			pw = strings.TrimRight(line, "\r\n")
		}
		if pw == "" {
			log.Fatal("empty password")
		}
		if len(pw) < 8 {
			fmt.Fprintf(os.Stderr, "warning: password is %d chars; the BBS requires >=8 for member-facing changes.\n", len(pw))
		}
		hash, herr := store.HashPassword(pw)
		if herr != nil {
			log.Fatalf("hash password: %v", herr)
		}

		// Create (or reuse) the user, set password, approve at the SysOp level.
		u, uerr := db.Users().ByHandle(handle)
		if uerr != nil {
			u, uerr = db.Users().Create(handle, "", "", "")
			if uerr != nil {
				log.Fatalf("create user: %v", uerr)
			}
		}
		if err := db.Users().SetPassword(u.ID, hash); err != nil {
			log.Fatalf("set password: %v", err)
		}
		if err := db.Users().Approve(u.ID, level); err != nil {
			log.Fatalf("approve: %v", err)
		}

		// Register the SSH key (idempotent: skip if this account already has it).
		alreadyHas := false
		if pk, _, _, _, perr := ssh.ParseAuthorizedKey([]byte(pubLine)); perr == nil {
			fp := ssh.FingerprintSHA256(pk)
			if active, e := db.Keys().Active(u.ID); e == nil {
				for _, k := range active {
					if k.Fingerprint == fp {
						alreadyHas = true
					}
				}
			}
		}
		if !alreadyHas {
			k, kerr := db.Keys().Add(u.ID, pubLine)
			if errors.Is(kerr, store.ErrKeyTaken) {
				log.Fatalf("that SSH key is already registered to another account")
			}
			if kerr != nil {
				log.Fatalf("add key: %v", kerr)
			}
			fmt.Printf("registered SSH key %s\n", k.Fingerprint)
		} else {
			fmt.Println("SSH key already registered to this account")
		}
		fmt.Printf("SysOp %q is ready at access level %d (password set, key registered).\n", handle, level)
		fmt.Println("Log in over SSH with that key + password — no onboarding token needed.")

	case "addkey":
		// addkey <handle> <pubkey-file|-> — register an ADDITIONAL SSH key on an
		// existing account (e.g. a second device). Unlike bootstrap it touches
		// neither the password nor the access level.
		if len(args) < 3 {
			log.Fatalf("usage: sysopctl addkey <handle> <pubkey-file|->")
		}
		handle := args[1]
		var keyBytes []byte
		if args[2] == "-" {
			keyBytes, err = io.ReadAll(os.Stdin)
		} else {
			keyBytes, err = os.ReadFile(args[2])
		}
		if err != nil {
			log.Fatalf("read public key: %v", err)
		}
		pubLine := strings.TrimSpace(string(keyBytes))
		if err := store.ValidatePublicKey(pubLine); err != nil {
			log.Fatalf("invalid SSH public key: %v", err)
		}
		u, uerr := db.Users().ByHandle(handle)
		if uerr != nil {
			log.Fatalf("no such user %q: %v", handle, uerr)
		}
		k, kerr := db.Keys().Add(u.ID, pubLine)
		if errors.Is(kerr, store.ErrKeyTaken) {
			log.Fatal("that SSH key is already registered (to this or another account)")
		}
		if kerr != nil {
			log.Fatalf("add key: %v", kerr)
		}
		fmt.Printf("registered SSH key %s on %q.\n", k.Fingerprint, handle)

	default:
		log.Fatalf("unknown command %q", args[0])
	}
}
