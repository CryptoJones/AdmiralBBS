package tests

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"admiralbbs/src/store"
)

func TestFileLibrary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bbs.db")
	s, err := store.Open(path, testVault(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureSeedFileAreas(); err != nil {
		t.Fatal(err)
	}
	uploader, _ := s.Users().Create("zerocool", "", "", "")

	areas, _ := s.FileAreas().Visible(50)
	if len(areas) != 1 {
		t.Fatalf("seed file areas = %d, want 1", len(areas))
	}

	body := []byte("SECRETFILE: the dorsal venous arch is on the back of the hand")
	f, err := s.Files().Add(areas[0].ID, uploader.ID, "notes.txt", "study notes", body)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if f.SizeBytes != int64(len(body)) {
		t.Fatalf("size = %d, want %d", f.SizeBytes, len(body))
	}

	// Download round-trips and bumps the counter.
	got, err := s.Files().Content(f.ID)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatalf("content round-trip failed: %v", err)
	}
	again, _ := s.Files().ByID(f.ID)
	if again.DownloadCount != 1 {
		t.Fatalf("download_count = %d, want 1", again.DownloadCount)
	}

	// Oversize rejected.
	if _, err := s.Files().Add(areas[0].ID, uploader.ID, "big", "", make([]byte, store.MaxFileBytes+1)); err != store.ErrTooLarge {
		t.Fatalf("oversize: want ErrTooLarge, got %v", err)
	}

	// Path-traversal-proof: a hostile filename does NOT create a file outside
	// the files dir; the blob is stored by row id.
	evil, err := s.Files().Add(areas[0].ID, uploader.ID, "../../../../etc/passwd", "nope", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	filesDir := filepath.Join(dir, "files")
	if _, statErr := os.Stat(filepath.Join(filesDir, "../../../../etc/passwd")); statErr == nil {
		t.Fatal("hostile filename escaped the files dir")
	}
	if _, statErr := os.Stat(filepath.Join(filesDir, "%d.bin")); statErr == nil {
		t.Fatal("blob path used literal filename, not id")
	}
	// the evil entry's blob is a normal <id>.bin
	if _, statErr := os.Stat(filepath.Join(filesDir, fmt.Sprintf("%d.bin", evil.ID))); statErr != nil {
		t.Fatalf("id-based blob missing: %v", statErr)
	}

	// Blob is ciphertext at rest.
	s.Close()
	var blob []byte
	entries, _ := os.ReadDir(filesDir)
	for _, e := range entries {
		b, _ := os.ReadFile(filepath.Join(filesDir, e.Name()))
		blob = append(blob, b...)
	}
	if bytes.Contains(blob, []byte("SECRETFILE")) {
		t.Error("file content found in plaintext at rest")
	}
}
