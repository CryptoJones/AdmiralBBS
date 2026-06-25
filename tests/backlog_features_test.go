package tests

import (
	"path/filepath"
	"testing"

	"admiralbbs/src/audit"
	"admiralbbs/src/crypto"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
	"admiralbbs/src/transport"
)

// driveSession replays scripted keystrokes through fn against a fresh session.
func driveSession(t *testing.T, lg *audit.Logger, handle string, input string, fn func(*session.Session)) {
	t.Helper()
	c := newFakeConn(input, "ansi", transport.WindowSize{Cols: 80, Rows: 25})
	c.user = handle
	c.tr = "ssh"
	sess := session.New("s-"+handle, c, lg, nil)
	fn(sess)
	sess.Close()
}

func testLogger(t *testing.T, s *store.Store, v *crypto.Vault) *audit.Logger {
	t.Helper()
	lg, err := audit.New(filepath.Join(t.TempDir(), "audit.jsonl"), v, s.SessionLog())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lg.Close() })
	return lg
}

// Feature #3: a SysOp can award (and dock) points; the total persists.
func TestUserAddPoints(t *testing.T) {
	s, _ := openTestStore(t)
	u, _ := s.Users().Create("scorer", "h", "", "")

	if u.Points != 0 {
		t.Fatalf("new user should start at 0 points, got %d", u.Points)
	}
	if total, err := s.Users().AddPoints(u.ID, 25); err != nil || total != 25 {
		t.Fatalf("AddPoints(+25) = %d, %v; want 25", total, err)
	}
	if total, err := s.Users().AddPoints(u.ID, -10); err != nil || total != 15 {
		t.Fatalf("AddPoints(-10) = %d, %v; want 15", total, err)
	}
	got, _ := s.Users().ByID(u.ID)
	if got.Points != 15 {
		t.Fatalf("persisted points = %d, want 15", got.Points)
	}
}

// Feature #2: a SysOp creates a new board category from the board menu. With no
// step-up secret configured, the [N]ew flow creates the area directly.
func TestBoardMenuSysOpCreatesBoard(t *testing.T) {
	s, v := openTestStore(t)
	lg := testLogger(t, s, v)
	sysop, _ := s.Users().Create("op", "h", "", "")
	_ = s.Users().SetStatus(sysop.ID, store.StatusApproved, store.SysOpLevel)
	sysop, _ = s.Users().ByID(sysop.ID)

	before, _ := s.MessageAreas().Count()
	// n -> new board; name; description; min-level (blank=0); any-key; q to exit.
	driveSession(t, lg, "op", "n\nAnnouncements\nSysOp notices\n\n\nq\n", func(sess *session.Session) {
		if err := menu.RunBoards(sess, s, sysop, ""); err != nil {
			t.Fatalf("boards: %v", err)
		}
	})
	after, _ := s.MessageAreas().Count()
	if after != before+1 {
		t.Fatalf("board count %d -> %d, want +1", before, after)
	}
	areas, _ := s.MessageAreas().Visible(store.SysOpLevel)
	found := false
	for _, a := range areas {
		if a.Name == "Announcements" {
			found = true
		}
	}
	if !found {
		t.Fatal("new board 'Announcements' not visible")
	}
}

// Feature #2 (gate): when a SysOp step-up secret is configured, the new-board
// flow rejects a wrong password and accepts the right one.
func TestBoardMenuNewBoardPasswordGate(t *testing.T) {
	s, v := openTestStore(t)
	lg := testLogger(t, s, v)
	sysop, _ := s.Users().Create("op2", "h", "", "")
	_ = s.Users().SetStatus(sysop.ID, store.StatusApproved, store.SysOpLevel)
	sysop, _ = s.Users().ByID(sysop.ID)

	before, _ := s.MessageAreas().Count()
	// Wrong password: denied, any-key, quit. No board created.
	driveSession(t, lg, "op2", "n\nwrong\n\nq\n", func(sess *session.Session) {
		if err := menu.RunBoards(sess, s, sysop, "s3cret"); err != nil {
			t.Fatalf("boards: %v", err)
		}
	})
	if n, _ := s.MessageAreas().Count(); n != before {
		t.Fatalf("wrong password created a board: %d -> %d", before, n)
	}

	// Correct password: board created with the given min level.
	driveSession(t, lg, "op2", "n\ns3cret\nVIP\nMembers only\n50\n\nq\n", func(sess *session.Session) {
		if err := menu.RunBoards(sess, s, sysop, "s3cret"); err != nil {
			t.Fatalf("boards: %v", err)
		}
	})
	areas, _ := s.MessageAreas().Visible(store.SysOpLevel)
	var vip *store.MessageArea
	for _, a := range areas {
		if a.Name == "VIP" {
			vip = a
		}
	}
	if vip == nil {
		t.Fatal("correct password did not create the VIP board")
	}
	if vip.MinAccessLevel != 50 {
		t.Fatalf("VIP min access level = %d, want 50", vip.MinAccessLevel)
	}
}

// Feature #4: at the mail "To:" prompt, "?" opens the member directory so the
// sender can pick a recipient by number without knowing the handle.
func TestMailToPromptLookup(t *testing.T) {
	s, v := openTestStore(t)
	lg := testLogger(t, s, v)
	alice, _ := s.Users().Create("alice", "h", "", "")
	_ = s.Users().SetStatus(alice.ID, store.StatusApproved, 50)
	alice, _ = s.Users().ByID(alice.ID)
	bob, _ := s.Users().Create("bob", "h", "", "")
	_ = s.Users().SetStatus(bob.ID, store.StatusApproved, 50)

	// c -> compose; ? -> look up; 1 -> pick bob; subject; body; '.'; q -> exit.
	driveSession(t, lg, "alice", "c\n?\n1\nHi Bob\nhello there\n.\nq\n", func(sess *session.Session) {
		if err := menu.RunMail(sess, s, alice); err != nil {
			t.Fatalf("mail: %v", err)
		}
	})
	if n, _ := s.Mail().UnreadCount(bob.ID); n != 1 {
		t.Fatalf("bob unread = %d, want 1 (lookup-picked recipient)", n)
	}
	inbox, _ := s.Mail().Inbox(bob.ID)
	if len(inbox) != 1 || inbox[0].Subject != "Hi Bob" {
		t.Fatalf("bob inbox = %+v, want one mail 'Hi Bob'", inbox)
	}
}
