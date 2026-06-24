package tests

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"admiralbbs/src/audit"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/transport"
)

// End-to-end: with more messages than fit on a page, the inbox shows a page
// footer, the first page omits the oldest item, and [>] reveals it.
func TestMailPaginationInMenu(t *testing.T) {
	s, v := openTestStore(t)
	lg, err := audit.New(filepath.Join(t.TempDir(), "audit.jsonl"), v, s.SessionLog())
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	alice, _ := s.Users().Create("alice", "", "", "")
	sender, _ := s.Users().Create("sender", "", "", "")

	// 20 messages > one 15-row page. Subjects are ordinal so we can locate them;
	// inbox is newest-first, so SUBJ-00 is the oldest (last page).
	for i := 0; i < 20; i++ {
		if _, err := s.Mail().Send(sender.ID, alice.ID, fmt.Sprintf("SUBJ-%02d", i), "x"); err != nil {
			t.Fatal(err)
		}
	}

	drive := func(input string) string {
		c := newFakeConn(input, "ansi", transport.WindowSize{Cols: 80, Rows: 25})
		c.user, c.tr = "alice", "ssh"
		sess := session.New("s-alice", c, lg, nil)
		if err := menu.RunMail(sess, s, alice); err != nil {
			t.Fatalf("mail menu: %v", err)
		}
		sess.Close()
		return c.output()
	}

	// Page 1: footer present, oldest (SUBJ-00) not yet shown.
	p1 := drive("q\n")
	if !strings.Contains(p1, "page 1/2") {
		t.Errorf("page-1 footer missing; got:\n%s", lastLines(p1))
	}
	if strings.Contains(p1, "SUBJ-00") {
		t.Error("oldest message should not appear on page 1")
	}

	// Advance one page, then quit: the oldest message is now visible.
	p2 := drive(">\nq\n")
	if !strings.Contains(p2, "SUBJ-00") {
		t.Error("oldest message should appear after paging forward")
	}
}

func lastLines(s string) string {
	parts := strings.Split(strings.TrimRight(s, "\r\n"), "\n")
	if len(parts) > 6 {
		parts = parts[len(parts)-6:]
	}
	return strings.Join(parts, "\n")
}
