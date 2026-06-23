package tests

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"admiralbbs/src/audit"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
	"admiralbbs/src/transport"
)

// The Telnet apply flow creates a pending user + key + application, and collects
// no password (set later over SSH).
func TestRunApplyCreatesPendingApplication(t *testing.T) {
	s, v := openTestStore(t)
	lg, err := audit.New(filepath.Join(t.TempDir(), "a.jsonl"), v)
	if err != nil {
		t.Fatal(err)
	}

	key := strings.TrimSpace(genSSHKey(t))
	// handle, one key, blank line to finish keys, contact, note
	input := "alice\n" + key + "\n\n" + "alice@pgp.example\n" + "here for the door games\n"
	conn := newFakeConn(input, "ansi", transport.WindowSize{Cols: 80})
	sess := session.New("s-1", conn, lg, func() time.Time { return time.Unix(1_700_000_000, 0) })

	if err := menu.RunApply(sess, s.Users(), s.Memberships(), s.Keys()); err != nil {
		t.Fatalf("apply: %v", err)
	}
	sess.Close()

	u, err := s.Users().ByHandle("alice")
	if err != nil {
		t.Fatalf("applicant not created: %v", err)
	}
	if u.Status != store.StatusPending {
		t.Fatalf("status = %s, want pending", u.Status)
	}
	if u.PasswordHash != "" {
		t.Fatalf("no password should be set over Telnet, got %q", u.PasswordHash)
	}
	if act, _ := s.Keys().Active(u.ID); len(act) != 1 {
		t.Fatalf("active keys = %d, want 1", len(act))
	}
	if pend, _ := s.Memberships().Pending(); len(pend) != 1 {
		t.Fatalf("pending applications = %d, want 1", len(pend))
	}
}
