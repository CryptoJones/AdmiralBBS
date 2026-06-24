// Command sysopctl is the operator's bootstrap/management tool, run on the BBS
// host (it needs ADMIRALBBS_KEY). It exists primarily to create the FIRST SysOp
// — without it, the access-gated control panel is unreachable on a fresh BBS.
//
//	ADMIRALBBS_KEY=... sysopctl -db data/admiralbbs.db -salt data/key.salt list
//	ADMIRALBBS_KEY=... sysopctl approve <handle> [level]   # approve + print token
//	ADMIRALBBS_KEY=... sysopctl promote <handle> [level]   # set status+level directly
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"admiralbbs/src/crypto"
	"admiralbbs/src/store"
)

func main() {
	dbPath := flag.String("db", "admiralbbs.db", "SQLite database path")
	saltPath := flag.String("salt", "key.salt", "KDF salt path")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("usage: sysopctl [-db ...] [-salt ...] list | approve <handle> [level] | promote <handle> [level]")
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
		level := 100
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

	default:
		log.Fatalf("unknown command %q", args[0])
	}
}
