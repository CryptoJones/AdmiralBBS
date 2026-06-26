package tests

import (
	"bytes"
	"encoding/base64"
	"path/filepath"
	"testing"

	"admiralbbs/src/audit"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
	"admiralbbs/src/transport"
)

// TestBase64Upload checks a binary file uploaded via the [B]ase64 paste mode is
// stored byte-for-byte (#8).
func TestBase64Upload(t *testing.T) {
	s, v := openTestStore(t)
	if err := s.EnsureSeedFileAreas(); err != nil {
		t.Fatal(err)
	}
	lg, err := audit.New(filepath.Join(t.TempDir(), "audit.jsonl"), v, s.SessionLog())
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	user, _ := s.Users().Create("dora", "x", "", "")
	_ = s.Users().Approve(user.ID, 50)
	user.AccessLevel = 50

	// Binary payload with NULs and high bytes — not representable as paste text.
	payload := []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 'P', 'K', 0x03, 0x04, 0x00}
	b64 := base64.StdEncoding.EncodeToString(payload)

	// area 1 -> [U]pload -> filename -> desc -> [B]ase64 -> b64 -> . -> [Q]uit menu
	input := "1\nu\nblob.bin\na binary\nb\n" + b64 + "\n.\nq\nq\n"
	c := newFakeConn(input, "ansi", transport.WindowSize{Cols: 80, Rows: 25})
	c.user = "dora"
	c.tr = "ssh"
	sess := session.New("s-dora", c, lg, nil)
	if err := menu.RunFiles(sess, s, user, ""); err != nil {
		t.Fatalf("RunFiles: %v", err)
	}
	sess.Close()

	areas, _ := s.FileAreas().Visible(user.AccessLevel)
	files, _ := s.Files().ListByArea(areas[0].ID)
	if len(files) != 1 || files[0].Filename != "blob.bin" {
		t.Fatalf("base64 upload not stored: %+v", files)
	}
	got, err := s.Files().Content(files[0].ID)
	if err != nil {
		t.Fatalf("read content: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("stored bytes differ from the binary payload:\n got %v\nwant %v", got, payload)
	}
	_ = store.MaxFileBytes
}
