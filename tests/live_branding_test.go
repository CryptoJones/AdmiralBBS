package tests

import (
	"strings"
	"testing"

	"admiralbbs/src/doors"
	"admiralbbs/src/menu"
)

// Editing the BBS name/tagline (and MOTD) must take effect on the next render of
// an already-running session — not require a relog. The main menu carries a
// Refresh hook that re-reads settings each render.
func TestMenuBrandingUpdatesLive(t *testing.T) {
	s, _ := openTestStore(t)
	s.Settings().Set("bbs_name", "OldNet")
	s.Settings().Set("tagline", "old tagline")
	u, _ := s.Users().Create("alice", "", "", "")

	m := menu.Member(s, u, "", "", doors.Opts{}, 1, "", nil, "")
	if m.Refresh == nil {
		t.Fatal("main menu should have a Refresh hook")
	}
	if !strings.Contains(m.Title, "OldNet") {
		t.Fatalf("initial title = %q", m.Title)
	}

	// SysOp changes the marketing info mid-session.
	s.Settings().Set("bbs_name", "NewNet")
	s.Settings().Set("tagline", "jacked in and dangerous")

	// The very next render (simulated by Refresh) reflects it — no relog.
	m.Refresh()
	if !strings.Contains(m.Title, "NewNet") {
		t.Fatalf("title after live edit = %q, want NewNet", m.Title)
	}
	joined := strings.Join(m.Banner, " ")
	if !strings.Contains(joined, "jacked in and dangerous") {
		t.Fatalf("banner didn't pick up the new tagline: %q", joined)
	}

	// MOTD item appears live once a MOTD is set.
	hasO := func() bool {
		for _, it := range m.Items {
			if it.Key == 'O' {
				return true
			}
		}
		return false
	}
	if hasO() {
		t.Fatal("no MOTD set yet, [O] should be absent")
	}
	s.Settings().Set("motd", "read me")
	m.Refresh()
	if !hasO() {
		t.Fatal("[O] Message of the Day should appear live after a MOTD is set")
	}
}
