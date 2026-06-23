package tests

import (
	"path/filepath"
	"testing"
	"time"

	"admiralbbs/src/audit"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
	"admiralbbs/src/transport"
)

// sshConn builds a fake SSH-transport conn for a given handle + scripted input.
func sshConn(handle, keys string) *fakeConn {
	c := newFakeConn(keys, "ansi", transport.WindowSize{Cols: 80})
	c.user = handle
	c.tr = "ssh"
	return c
}

func TestLoginOnboardThenVerify(t *testing.T) {
	s, v := openTestStore(t)
	lg, err := audit.New(filepath.Join(t.TempDir(), "a.jsonl"), v)
	if err != nil {
		t.Fatal(err)
	}

	// Approved user, no password yet, with a one-time token.
	u, _ := s.Users().Create("alice", "", "", "")
	if err := s.Users().Approve(u.ID, 50); err != nil {
		t.Fatal(err)
	}
	tok, err := s.Tokens().Issue(u.ID)
	if err != nil {
		t.Fatal(err)
	}

	// First login: token, then password twice.
	conn := sshConn("alice", tok+"\nhunter2pw\nhunter2pw\n")
	sess := session.New("s-1", conn, lg, func() time.Time { return time.Unix(1_700_000_000, 0) })
	got, ok := menu.RunLogin(sess, s)
	sess.Close()
	if !ok || got == nil {
		t.Fatalf("onboarding login failed")
	}

	// Password is now set in the DB.
	reread, _ := s.Users().ByHandle("alice")
	if reread.PasswordHash == "" {
		t.Fatal("password not persisted after onboarding")
	}

	// Subsequent login with the correct password succeeds.
	conn2 := sshConn("alice", "hunter2pw\n")
	sess2 := session.New("s-2", conn2, lg, func() time.Time { return time.Unix(1_700_000_100, 0) })
	if _, ok := menu.RunLogin(sess2, s); !ok {
		t.Fatal("verify login with correct password failed")
	}
	sess2.Close()
}

func TestLoginWrongPasswordFails(t *testing.T) {
	s, v := openTestStore(t)
	lg, _ := audit.New(filepath.Join(t.TempDir(), "a.jsonl"), v)

	u, _ := s.Users().Create("carol", "", "", "")
	s.Users().Approve(u.ID, 50)
	hash, _ := store.HashPassword("correct-horse")
	s.Users().SetPassword(u.ID, hash)

	conn := sshConn("carol", "wrongpassword\n")
	sess := session.New("s-3", conn, lg, func() time.Time { return time.Unix(1_700_000_000, 0) })
	if _, ok := menu.RunLogin(sess, s); ok {
		t.Fatal("login succeeded with wrong password")
	}
	sess.Close()
}
