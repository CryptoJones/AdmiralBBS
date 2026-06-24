// Command rekey rotates the AdmiralBBS master key: it re-encrypts every sealed
// DB field, every file-library blob, and the audit trail from the OLD key to a
// NEW key, then writes the new KDF salt. Run with the daemon STOPPED.
//
//	ADMIRALBBS_KEY=<old> ADMIRALBBS_NEW_KEY=<new> \
//	  rekey -db data/admiralbbs.db -audit data/audit.jsonl -salt data/key.salt
//
// Secrets come ONLY from the environment — never flags, never chat. Back up the
// data dir first (see docs/OPERATIONS.md).
package main

import (
	"flag"
	"log"
	"os"

	"admiralbbs/src/audit"
	"admiralbbs/src/crypto"
	"admiralbbs/src/store"
)

func main() {
	dbPath := flag.String("db", "admiralbbs.db", "SQLite database path")
	auditPath := flag.String("audit", "audit.jsonl", "audit log path")
	saltPath := flag.String("salt", "key.salt", "KDF salt path (read old, write new)")
	flag.Parse()

	oldSecret := os.Getenv("ADMIRALBBS_KEY")
	newSecret := os.Getenv("ADMIRALBBS_NEW_KEY")
	if oldSecret == "" || newSecret == "" {
		log.Fatal("set ADMIRALBBS_KEY (old) and ADMIRALBBS_NEW_KEY (new) in the env")
	}
	if oldSecret == newSecret {
		log.Fatal("new key must differ from the old key")
	}

	oldSalt, err := os.ReadFile(*saltPath)
	if err != nil {
		log.Fatalf("read old salt: %v", err)
	}
	oldVault, err := crypto.NewVault([]byte(oldSecret), oldSalt)
	if err != nil {
		log.Fatalf("old vault: %v", err)
	}
	defer oldVault.Close()

	newSalt, err := crypto.GenerateSalt()
	if err != nil {
		log.Fatalf("new salt: %v", err)
	}
	newVault, err := crypto.NewVault([]byte(newSecret), newSalt)
	if err != nil {
		log.Fatalf("new vault: %v", err)
	}
	defer newVault.Close()

	log.Printf("re-encrypting database + blobs (%s)...", *dbPath)
	if err := store.RekeyDB(*dbPath, oldVault, newVault); err != nil {
		log.Fatalf("rekey db: %v (no salt written; old key still valid)", err)
	}
	log.Printf("re-encrypting audit trail (%s)...", *auditPath)
	if err := audit.Rekey(*auditPath, oldVault, newVault); err != nil {
		log.Fatalf("rekey audit: %v (DB already rotated to NEW key — re-run audit step)", err)
	}

	// Only now commit the new salt, so a mid-run failure leaves the old salt
	// (and thus the recoverable old key) in place.
	if err := os.WriteFile(*saltPath, newSalt, 0o600); err != nil {
		log.Fatalf("write new salt: %v", err)
	}
	log.Printf("done — restart the daemon with ADMIRALBBS_KEY set to the NEW key")
}
