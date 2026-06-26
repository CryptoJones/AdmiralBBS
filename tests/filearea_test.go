package tests

import (
	"path/filepath"
	"testing"

	"admiralbbs/src/audit"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
	"admiralbbs/src/transport"
)

// TestSysOpCreatesFileArea drives the File menu as a SysOp and creates a new
// File Area via [N]ew area (#6).
func TestSysOpCreatesFileArea(t *testing.T) {
	s, v := openTestStore(t)
	if err := s.EnsureSeedFileAreas(); err != nil {
		t.Fatal(err)
	}
	lg, err := audit.New(filepath.Join(t.TempDir(), "audit.jsonl"), v, s.SessionLog())
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	boss, _ := s.Users().Create("boss", "x", "", "")
	_ = s.Users().Approve(boss.ID, store.SysOpLevel)
	boss.AccessLevel = store.SysOpLevel // the SysOp gate reads the in-memory user

	before, _ := s.FileAreas().Count()

	// [N]ew area -> name -> min level -> ack key -> quit. No SysOp step-up pass.
	c := newFakeConn("n\nArchive Files\n0\nx\nq\n", "ansi", transport.WindowSize{Cols: 80, Rows: 25})
	c.user = "boss"
	c.tr = "ssh"
	sess := session.New("s-boss", c, lg, nil)
	if err := menu.RunFiles(sess, s, boss, ""); err != nil {
		t.Fatalf("RunFiles: %v", err)
	}
	sess.Close()

	after, _ := s.FileAreas().Count()
	if after != before+1 {
		t.Fatalf("SysOp [N]ew area should create one area: %d -> %d", before, after)
	}
	areas, _ := s.FileAreas().Visible(store.SysOpLevel)
	found := false
	for _, a := range areas {
		if a.Name == "Archive Files" {
			found = true
		}
	}
	if !found {
		t.Errorf("new area 'Archive Files' not found among %d areas", len(areas))
	}
}
