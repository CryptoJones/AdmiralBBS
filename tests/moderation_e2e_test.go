package tests

import (
	"path/filepath"
	"strings"
	"testing"

	"admiralbbs/src/audit"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/transport"
)

// End-to-end through the real mail menu: a blocked sender's mail is absent from
// the inbox the member actually sees, while an unblocked sender's is present.
func TestBlockHidesMailInMenu(t *testing.T) {
	s, v := openTestStore(t)
	lg, err := audit.New(filepath.Join(t.TempDir(), "audit.jsonl"), v, s.SessionLog())
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	alice, _ := s.Users().Create("alice", "", "", "")
	bob, _ := s.Users().Create("bob", "", "", "")
	carol, _ := s.Users().Create("carol", "", "", "")

	if _, err := s.Mail().Send(bob.ID, alice.ID, "FROM-BOB-HIDDEN", "spam"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Mail().Send(carol.ID, alice.ID, "FROM-CAROL-VISIBLE", "hi"); err != nil {
		t.Fatal(err)
	}
	if err := s.Blocks().Block(alice.ID, bob.ID); err != nil {
		t.Fatal(err)
	}

	c := newFakeConn("q\n", "ansi", transport.WindowSize{Cols: 80, Rows: 25})
	c.user = "alice"
	c.tr = "ssh"
	sess := session.New("s-alice", c, lg, nil)
	if err := menu.RunMail(sess, s, alice); err != nil {
		t.Fatalf("mail menu: %v", err)
	}
	sess.Close()

	out := c.output()
	if strings.Contains(out, "FROM-BOB-HIDDEN") {
		t.Error("blocked sender's mail appeared in the inbox")
	}
	if !strings.Contains(out, "FROM-CAROL-VISIBLE") {
		t.Error("unblocked sender's mail was missing from the inbox")
	}
}

// Sanity: the report filed contextually from a read view shape (reporter,
// target, context) lands in the SysOp open queue.
func TestReportFromContextLandsInQueue(t *testing.T) {
	s, _ := openTestStore(t)
	alice, _ := s.Users().Create("alice", "", "", "")
	bob, _ := s.Users().Create("bob", "", "", "")
	if _, err := s.Reports().File(alice.ID, bob.ID, "board post #3", "harassment"); err != nil {
		t.Fatal(err)
	}
	open, _ := s.Reports().Open()
	if len(open) != 1 || open[0].ReporterID != alice.ID || open[0].TargetID != bob.ID {
		t.Fatalf("report did not land in queue: %+v", open)
	}
}
