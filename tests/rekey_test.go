package tests

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"admiralbbs/src/audit"
	"admiralbbs/src/crypto"
	"admiralbbs/src/store"
)

func TestRekeyRotatesEverything(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "bbs.db")
	auditPath := filepath.Join(dir, "audit.jsonl")

	vA, err := crypto.NewVault([]byte("old-key-AAAA"), []byte("0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}

	// Seed data + audit under key A.
	s, err := store.Open(dbPath, vA)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := s.Users().Create("alice", "", "Alice Real", "alice@secret.example")
	area, _ := s.FileAreas().Create("General", 0)
	f, _ := s.Files().Add(area.ID, u.ID, "notes.txt", "study", []byte("SECRET-BLOB-BODY"))
	lg, _ := audit.New(auditPath, vA)
	lg.Emit(audit.Event{Type: audit.TypeActivity, SessionID: "s1", Detail: "secret-detail", Time: time.Now()})
	lg.Close()
	s.Close()

	// Rotate A -> B.
	vB, err := crypto.NewVault([]byte("new-key-BBBB"), []byte("fedcba9876543210"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RekeyDB(dbPath, vA, vB); err != nil {
		t.Fatalf("RekeyDB: %v", err)
	}
	if err := audit.Rekey(auditPath, vA, vB); err != nil {
		t.Fatalf("audit.Rekey: %v", err)
	}

	// Everything reads under the NEW key.
	sB, err := store.Open(dbPath, vB)
	if err != nil {
		t.Fatal(err)
	}
	got, err := sB.Users().ByHandle("alice")
	if err != nil || got.Email != "alice@secret.example" || got.RealName != "Alice Real" {
		t.Fatalf("PII not readable under new key: %+v err=%v", got, err)
	}
	blob, err := sB.Files().Content(f.ID)
	if err != nil || !bytes.Equal(blob, []byte("SECRET-BLOB-BODY")) {
		t.Fatalf("blob not readable under new key: %v", err)
	}
	evs, err := audit.ReadAll(auditPath, vB)
	if err != nil || len(evs) != 1 || evs[0].Detail != "secret-detail" {
		t.Fatalf("audit not readable/verified under new key: %v %+v", err, evs)
	}
	sB.Close()

	// The OLD key can no longer read the rotated data.
	sA, _ := store.Open(dbPath, vA)
	if _, err := sA.Users().ByHandle("alice"); err == nil {
		t.Fatal("old key still decrypts after rekey — rotation incomplete")
	}
	sA.Close()
	if _, err := audit.ReadAll(auditPath, vA); err == nil {
		t.Fatal("old key still verifies audit after rekey")
	}
}
