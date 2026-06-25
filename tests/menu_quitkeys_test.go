package tests

import (
	"strings"
	"testing"

	"admiralbbs/src/doors"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
	"admiralbbs/src/transport"
)

// itemKey reports whether the menu has an item bound to key, and its label.
func itemKey(m *menu.Menu, key byte) (bool, string) {
	for _, it := range m.Items {
		if it.Key == key {
			return true, it.Label
		}
	}
	return false, ""
}

// The SysOp Control Panel must be on [S], never on [X] — X and Q are reserved
// quit keys. A non-SysOp sees neither S nor X.
func TestSysOpMenuKeyMovedOffX(t *testing.T) {
	s, _ := openTestStore(t)

	sysop, _ := s.Users().Create("op", "", "", "")
	_ = s.Users().SetStatus(sysop.ID, store.StatusApproved, store.SysOpLevel)
	sysop, _ = s.Users().ByID(sysop.ID)
	m := menu.Member(s, sysop, "", "", doors.Opts{}, 1, "", nil, "", nil)

	if hasX, _ := itemKey(m, 'X'); hasX {
		t.Fatal("no menu item may be bound to the reserved key [X]")
	}
	hasS, label := itemKey(m, 'S')
	if !hasS {
		t.Fatal("SysOp Control Panel should be bound to [S]")
	}
	if label != "SysOp Control Panel" {
		t.Fatalf("[S] label = %q, want SysOp Control Panel", label)
	}

	member, _ := s.Users().Create("bob", "", "", "")
	_ = s.Users().SetStatus(member.ID, store.StatusApproved, 50)
	member, _ = s.Users().ByID(member.ID)
	mm := menu.Member(s, member, "", "", doors.Opts{}, 1, "", nil, "", nil)
	if hasS, _ := itemKey(mm, 'S'); hasS {
		t.Fatal("a regular member must not see the SysOp [S] item")
	}
}

// The main menu logs off when a caller presses a reserved quit key (X or Q),
// even though the visible quit item is [G] Goodbye.
func TestMainMenuQuitsOnXorQ(t *testing.T) {
	for _, key := range []string{"x", "q", "X", "Q"} {
		s, v := openTestStore(t)
		lg := testLogger(t, s, v)
		u, _ := s.Users().Create("alice", "", "", "")
		_ = s.Users().SetStatus(u.ID, store.StatusApproved, 50)
		u, _ = s.Users().ByID(u.ID)

		c := newFakeConn(key+"\n", "ansi", transport.WindowSize{Cols: 80, Rows: 25})
		c.user, c.tr = "alice", "ssh"
		sess := session.New("s-quit", c, lg, nil)
		m := menu.Member(s, u, "", "", doors.Opts{}, 1, "", nil, "", nil)
		if err := m.Run(sess); err != nil {
			t.Fatalf("key %q: clean logoff should return nil, got %v", key, err)
		}
		// The goodbye banner proves the logoff action actually fired (not just EOF).
		if !strings.Contains(c.output(), "Thanks for calling") {
			t.Fatalf("key %q did not trigger logoff (no goodbye banner)", key)
		}
		sess.Close()
	}
}
