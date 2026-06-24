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

// End-to-end: a visitor sees the "new" markers on first entry; after browsing
// the area (which advances the read pointer on exit), a second visit shows none.
func TestBoardNewMarkersInMenu(t *testing.T) {
	s, v := openTestStore(t)
	if err := s.EnsureSeedAreas(); err != nil {
		t.Fatal(err)
	}
	lg, err := audit.New(filepath.Join(t.TempDir(), "audit.jsonl"), v, s.SessionLog())
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	areas, _ := s.MessageAreas().Visible(50)
	author, _ := s.Users().Create("author", "", "", "")
	visitor, _ := s.Users().Create("visitor", "", "", "")
	s.Messages().Post(areas[0].ID, author.ID, nil, "ALPHA", "b")
	s.Messages().Post(areas[0].ID, author.ID, nil, "BETA", "b")

	drive := func(input string) string {
		c := newFakeConn(input, "ansi", transport.WindowSize{Cols: 80, Rows: 25})
		c.user, c.tr = "visitor", "ssh"
		sess := session.New("s-visitor", c, lg, nil)
		if err := menu.RunBoards(sess, s, visitor); err != nil {
			t.Fatalf("boards: %v", err)
		}
		sess.Close()
		return c.output()
	}

	// First visit: enter area 1, quit area (advances pointer), quit boards.
	first := drive("1\nq\nq\n")
	if !strings.Contains(first, "new") {
		t.Errorf("first visit should show a 'new' count; got:\n%s", first)
	}

	// Second visit: just view the area list and quit. Nothing should be new.
	second := drive("q\n")
	if strings.Contains(second, "new") {
		t.Errorf("second visit should show nothing new; got:\n%s", second)
	}
}
