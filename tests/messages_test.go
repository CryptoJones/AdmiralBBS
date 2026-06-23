package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"admiralbbs/src/store"
)

func TestMessageAreasSeedAndVisibility(t *testing.T) {
	s, _ := openTestStore(t)
	if err := s.EnsureSeedAreas(); err != nil {
		t.Fatal(err)
	}
	// Idempotent.
	if err := s.EnsureSeedAreas(); err != nil {
		t.Fatal(err)
	}
	n, _ := s.MessageAreas().Count()
	if n != 2 {
		t.Fatalf("seed areas = %d, want 2", n)
	}

	// A restricted area is hidden from low access.
	if _, err := s.MessageAreas().Create("SysOps Only", "staff", 80); err != nil {
		t.Fatal(err)
	}
	vis, _ := s.MessageAreas().Visible(50)
	for _, a := range vis {
		if a.Name == "SysOps Only" {
			t.Fatal("restricted area visible to access level 50")
		}
	}
	if got, _ := s.MessageAreas().Visible(100); len(got) != 3 {
		t.Fatalf("sysop should see 3 areas, saw %d", len(got))
	}
}

func TestMessagePostThreadAndEncryption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bbs.db")
	s, err := store.Open(path, testVault(t))
	if err != nil {
		t.Fatal(err)
	}
	author, _ := s.Users().Create("zerocool", "", "", "")
	area, _ := s.MessageAreas().Create("General", "", 0)

	top, err := s.Messages().Post(area.ID, author.ID, nil, "Hello", "SECRETBODY content here")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Messages().Post(area.ID, author.ID, &top.ID, "re: Hello", "a reply body"); err != nil {
		t.Fatal(err)
	}

	thread, _ := s.Messages().Thread(area.ID)
	if len(thread) != 1 {
		t.Fatalf("top-level messages = %d, want 1 (reply must not be top-level)", len(thread))
	}
	if thread[0].Subject != "Hello" || thread[0].Body != "SECRETBODY content here" {
		t.Fatalf("round-trip failed: %+v", thread[0])
	}
	replies, _ := s.Messages().Replies(top.ID)
	if len(replies) != 1 || replies[0].Body != "a reply body" {
		t.Fatalf("replies wrong: %+v", replies)
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
	if bytes.Contains(blob, []byte("SECRETBODY")) {
		t.Error("message body found in plaintext at rest")
	}
}
