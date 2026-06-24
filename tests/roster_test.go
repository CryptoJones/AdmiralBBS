package tests

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"admiralbbs/src/audit"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/transport"
)

func TestRosterJoinListLeave(t *testing.T) {
	base := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	r := session.NewRoster(func() time.Time { return base })

	if r.Count() != 0 {
		t.Fatal("new roster should be empty")
	}
	r.Join(3, "carol", "203.0.113.9", "ssh")
	r.Join(1, "alice", "203.0.113.7", "ssh")
	r.Join(2, "bob", "203.0.113.8", "telnet")

	list := r.List()
	if len(list) != 3 {
		t.Fatalf("online = %d, want 3", len(list))
	}
	// Sorted by node number.
	if list[0].Node != 1 || list[1].Node != 2 || list[2].Node != 3 {
		t.Fatalf("not sorted by node: %+v", list)
	}
	if list[0].Handle != "alice" || list[0].Transport != "ssh" {
		t.Fatalf("node 1 fields wrong: %+v", list[0])
	}

	r.Leave(2)
	if r.Count() != 2 {
		t.Fatalf("after leave count = %d, want 2", r.Count())
	}
	for _, o := range r.List() {
		if o.Node == 2 {
			t.Fatal("node 2 still listed after leave")
		}
	}
}

// End-to-end: the who's-online menu renders the live roster.
func TestWhosOnlineMenuRendersRoster(t *testing.T) {
	r := session.NewRoster(nil)
	r.Join(1, "alice", "203.0.113.7", "ssh")
	r.Join(2, "bob", "203.0.113.8", "telnet")

	s, v := openTestStore(t)
	lg, err := audit.New(filepath.Join(t.TempDir(), "audit.jsonl"), v, s.SessionLog())
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	c := newFakeConn("\n", "ansi", transport.WindowSize{Cols: 80, Rows: 25})
	c.user, c.tr = "alice", "ssh"
	sess := session.New("s-alice", c, lg, nil)
	if err := menu.RunWhosOnline(sess, r); err != nil {
		t.Fatalf("whos-online: %v", err)
	}
	sess.Close()

	out := c.output()
	for _, want := range []string{"alice", "bob", "Node 1", "Node 2"} {
		if !strings.Contains(out, want) {
			t.Errorf("who's-online output missing %q", want)
		}
	}
}
