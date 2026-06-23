package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"admiralbbs/src/store"
)

// Sensitive fields must be ciphertext on disk; structural fields (handle) stay
// cleartext so the DB can index them (the encrypted volume covers them offline).
func TestSensitiveDataEncryptedAtRest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bbs.db")
	s, err := store.Open(path, testVault(t))
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.Users().Create("alice", "", "Top Secret Name", "topsecret@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Memberships().Apply(u.ID, "my private reason to join"); err != nil {
		t.Fatal(err)
	}
	s.DB().Exec("PRAGMA wal_checkpoint(TRUNCATE)") // flush WAL into the main file
	s.Close()

	// Concatenate every file in the data dir (db, -wal, -shm).
	var blob []byte
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		b, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		blob = append(blob, b...)
	}

	for _, secret := range []string{"Top Secret Name", "topsecret@example.com", "my private reason to join"} {
		if bytes.Contains(blob, []byte(secret)) {
			t.Errorf("plaintext %q found at rest", secret)
		}
	}
	if !bytes.Contains(blob, []byte("alice")) {
		t.Error("handle should be cleartext (indexable) but was not found on disk")
	}
}
