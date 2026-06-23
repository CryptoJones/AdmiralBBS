package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"admiralbbs/src/store"
)

func TestMailSendInboxReadAndEncryption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bbs.db")
	s, err := store.Open(path, testVault(t))
	if err != nil {
		t.Fatal(err)
	}
	alice, _ := s.Users().Create("alice", "", "", "")
	bob, _ := s.Users().Create("bob", "", "", "")

	if _, err := s.Mail().Send(alice.ID, bob.ID, "hi bob", "SECRETMAILBODY here"); err != nil {
		t.Fatal(err)
	}

	if n, _ := s.Mail().UnreadCount(bob.ID); n != 1 {
		t.Fatalf("unread = %d, want 1", n)
	}
	inbox, _ := s.Mail().Inbox(bob.ID)
	if len(inbox) != 1 || inbox[0].Subject != "hi bob" || inbox[0].ReadAt != nil {
		t.Fatalf("inbox wrong: %+v", inbox)
	}

	// Reading as the recipient marks it read and decrypts.
	got, err := s.Mail().Get(inbox[0].ID, bob.ID)
	if err != nil || got.Body != "SECRETMAILBODY here" || got.ReadAt == nil {
		t.Fatalf("read failed: %+v err=%v", got, err)
	}
	if n, _ := s.Mail().UnreadCount(bob.ID); n != 0 {
		t.Fatalf("unread after read = %d, want 0", n)
	}

	// A third party cannot read it.
	carol, _ := s.Users().Create("carol", "", "", "")
	if _, err := s.Mail().Get(inbox[0].ID, carol.ID); err != store.ErrNotFound {
		t.Fatalf("third party read: want ErrNotFound, got %v", err)
	}

	// Body is ciphertext at rest.
	s.DB().Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	s.Close()
	var blob []byte
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		b, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		blob = append(blob, b...)
	}
	if bytes.Contains(blob, []byte("SECRETMAILBODY")) {
		t.Error("mail body found in plaintext at rest")
	}
}
