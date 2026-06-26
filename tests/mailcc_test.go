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

func mailEnv(t *testing.T) (*store.Store, *audit.Logger) {
	t.Helper()
	s, v := openTestStore(t)
	if err := s.EnsureSeedAreas(); err != nil {
		t.Fatal(err)
	}
	lg, err := audit.New(filepath.Join(t.TempDir(), "audit.jsonl"), v, s.SessionLog())
	if err != nil {
		t.Fatal(err)
	}
	return s, lg
}

func approved(t *testing.T, s *store.Store, handle string) *store.User {
	t.Helper()
	u, err := s.Users().Create(handle, "x", "", "")
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Users().Approve(u.ID, 50)
	u.AccessLevel = 50
	return u
}

func runMailAs(t *testing.T, s *store.Store, lg *audit.Logger, u *store.User, input string) {
	t.Helper()
	c := newFakeConn(input, "ansi", transport.WindowSize{Cols: 80, Rows: 25})
	c.user = u.Handle
	c.tr = "ssh"
	sess := session.New("s-"+u.Handle, c, lg, nil)
	if err := menu.RunMail(sess, s, u); err != nil {
		t.Fatalf("RunMail: %v", err)
	}
	sess.Close()
}

// TestMailCC checks a message addressed To one member and CC another reaches
// both (#7).
func TestMailCC(t *testing.T) {
	s, lg := mailEnv(t)
	defer lg.Close()
	alice := approved(t, s, "alice")
	bob := approved(t, s, "bob")
	carol := approved(t, s, "carol")

	// compose -> To bob -> CC carol -> blank (finish CC) -> subject -> body -> . -> quit
	runMailAs(t, s, lg, alice, "c\nbob\ncarol\n\nHello\nbody line\n.\nq\n")

	if n, _ := s.Mail().UnreadCount(bob.ID); n != 1 {
		t.Errorf("To: recipient bob should have 1 unread, got %d", n)
	}
	if n, _ := s.Mail().UnreadCount(carol.ID); n != 1 {
		t.Errorf("CC: recipient carol should have 1 unread, got %d", n)
	}
}

// TestMailToWildcard checks the `*` search resolves a recipient by partial
// handle (#7).
func TestMailToWildcard(t *testing.T) {
	s, lg := mailEnv(t)
	defer lg.Close()
	alice := approved(t, s, "alice")
	bob := approved(t, s, "bob")

	// To "bo*" -> directory filtered to [bob] -> pick 1 -> blank CC -> send.
	runMailAs(t, s, lg, alice, "c\nbo*\n1\n\nHi\nbody\n.\nq\n")

	if n, _ := s.Mail().UnreadCount(bob.ID); n != 1 {
		t.Errorf("wildcard To: should reach bob, got %d unread", n)
	}
}
