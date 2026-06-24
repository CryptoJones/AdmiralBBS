package tests

import (
	"path/filepath"
	"strings"
	"testing"

	"admiralbbs/src/audit"
	"admiralbbs/src/doors"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
	"admiralbbs/src/transport"
)

func TestSettingsDefaultsAndOverride(t *testing.T) {
	s, _ := openTestStore(t)
	set := s.Settings()
	if set.BBSName() != store.DefaultBBSName {
		t.Fatalf("default name = %q", set.BBSName())
	}
	if set.MOTD() != "" {
		t.Fatalf("default MOTD should be empty, got %q", set.MOTD())
	}
	if err := set.Set("bbs_name", "NeonNet"); err != nil {
		t.Fatal(err)
	}
	if err := set.Set("motd", "welcome, choom"); err != nil {
		t.Fatal(err)
	}
	if set.BBSName() != "NeonNet" {
		t.Fatalf("override name = %q, want NeonNet", set.BBSName())
	}
	if set.MOTD() != "welcome, choom" {
		t.Fatalf("override motd = %q", set.MOTD())
	}
}

func newE2ESession(t *testing.T, s *store.Store, input string) *session.Session {
	t.Helper()
	v := testVault(t)
	lg, err := audit.New(filepath.Join(t.TempDir(), "audit.jsonl"), v, s.SessionLog())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lg.Close() })
	c := newFakeConn(input, "ansi", transport.WindowSize{Cols: 80, Rows: 25})
	c.user, c.tr = "alice", "ssh"
	return session.New("s-alice", c, lg, nil)
}

// The MOTD blocks until the caller presses SPACE.
func TestMOTDWaitsForSpace(t *testing.T) {
	s, _ := openTestStore(t)
	sess := newE2ESession(t, s, "qx ") // non-space keys, then SPACE
	c := sess
	if err := menu.ShowMOTD(c, "Hello world\nSecond line"); err != nil {
		t.Fatalf("ShowMOTD: %v", err)
	}
	// (it returned only after the SPACE — earlier 'q'/'x' didn't satisfy it)
}

// The main menu shows the SysOp-configured name + tagline.
func TestMenuShowsCustomBranding(t *testing.T) {
	s, _ := openTestStore(t)
	s.Settings().Set("bbs_name", "NeonNet")
	s.Settings().Set("tagline", "jack in, choom")
	u, _ := s.Users().Create("alice", "", "", "") // regular (no SysOp item)

	out, buf := sinkSession(t, s)
	m := menu.Member(s, u, "", "", doors.Opts{}, 1, "", nil, "")
	_ = m.Run(out)
	got := buf()
	if !strings.Contains(got, "NeonNet :: Main Menu") || !strings.Contains(got, "jack in, choom") {
		t.Errorf("menu missing custom branding; got:\n%s", got)
	}
}

// sinkSession returns a session whose only input is 'G' (logoff) and a getter
// for everything it wrote.
func sinkSession(t *testing.T, s *store.Store) (*session.Session, func() string) {
	t.Helper()
	v := testVault(t)
	lg, err := audit.New(filepath.Join(t.TempDir(), "audit.jsonl"), v, s.SessionLog())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lg.Close() })
	c := newFakeConn("G", "ansi", transport.WindowSize{Cols: 80, Rows: 25})
	c.user, c.tr = "alice", "ssh"
	sess := session.New("s-alice", c, lg, nil)
	return sess, func() string {
		raw := c.output()
		// strip ANSI for assertion
		var b strings.Builder
		skip := false
		for i := 0; i < len(raw); i++ {
			ch := raw[i]
			if ch == 0x1b {
				skip = true
				continue
			}
			if skip {
				if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
					skip = false
				}
				continue
			}
			b.WriteByte(ch)
		}
		return b.String()
	}
}
