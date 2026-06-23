package tests

import (
	"bytes"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"admiralbbs/src/audit"
	"admiralbbs/src/menu"
	"admiralbbs/src/session"
	"admiralbbs/src/transport"
)

// fakeConn is an in-memory transport.Conn: it replays scripted keystrokes and
// captures everything the BBS writes back.
type fakeConn struct {
	in       *bytes.Reader
	out      bytes.Buffer
	mu       sync.Mutex
	term     string
	ws       transport.WindowSize
	user     string
	tr       string
}

func newFakeConn(keys, term string, ws transport.WindowSize) *fakeConn {
	return &fakeConn{in: bytes.NewReader([]byte(keys)), term: term, ws: ws, tr: "telnet"}
}

func (c *fakeConn) Read(p []byte) (int, error) { return c.in.Read(p) }
func (c *fakeConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.out.Write(p)
}
func (c *fakeConn) Close() error { return nil }
func (c *fakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(203, 0, 113, 7), Port: 5555}
}
func (c *fakeConn) Transport() string                     { return c.tr }
func (c *fakeConn) TermType() string                      { return c.term }
func (c *fakeConn) WindowSize() transport.WindowSize      { return c.ws }
func (c *fakeConn) WindowChanges() <-chan transport.WindowSize { return nil }
func (c *fakeConn) Username() string                      { return c.user }
func (c *fakeConn) output() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.out.String()
}

// fixedClock returns base on the first call (session start) and base+90s after.
func fixedClock(base time.Time) session.Clock {
	var n int
	return func() time.Time {
		n++
		if n == 1 {
			return base
		}
		return base.Add(90 * time.Second)
	}
}

func readEvents(t *testing.T, path string) []audit.Event {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	var evs []audit.Event
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var e audit.Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("bad audit line %q: %v", line, err)
		}
		evs = append(evs, e)
	}
	return evs
}

func TestSpineEndToEnd_ANSI(t *testing.T) {
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	logger, err := audit.New(auditPath)
	if err != nil {
		t.Fatal(err)
	}

	// Caller: open Message Boards (M), press a key, then logoff (G).
	conn := newFakeConn("M G", "ansi", transport.WindowSize{Cols: 80, Rows: 25})
	s := session.New("s-000001", conn, logger, fixedClock(time.Unix(1_700_000_000, 0)))
	if err := menu.Demo("").Run(s); err != nil {
		t.Fatalf("menu run: %v", err)
	}
	s.Close()
	logger.Close()

	out := conn.output()
	if !strings.Contains(out, "Main Menu") {
		t.Errorf("menu title not rendered:\n%s", out)
	}
	if !strings.Contains(out, "NO CARRIER") {
		t.Errorf("logoff banner not shown")
	}
	if !bytes.Contains([]byte(out), []byte{0x1b}) {
		t.Errorf("ANSI session emitted no escape codes")
	}

	// Audit trail: connect, the M activity, logoff activity, disconnect.
	evs := readEvents(t, auditPath)
	if len(evs) < 4 {
		t.Fatalf("expected >=4 audit events, got %d: %+v", len(evs), evs)
	}
	if evs[0].Type != audit.TypeConnect || evs[0].RemoteIP != "203.0.113.7" {
		t.Errorf("first event wrong: %+v", evs[0])
	}
	last := evs[len(evs)-1]
	if last.Type != audit.TypeDisconnect {
		t.Errorf("last event not disconnect: %+v", last)
	}
	if last.Minutes != 1.5 {
		t.Errorf("disconnect minutes = %v, want 1.5", last.Minutes)
	}
	if !hasActivity(evs, "message-boards") || !hasActivity(evs, "logoff") {
		t.Errorf("expected message-boards and logoff activities, got %+v", evs)
	}
}

func TestSpineEndToEnd_BWNoEscapes(t *testing.T) {
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	logger, _ := audit.New(auditPath)
	conn := newFakeConn("G", "dumb", transport.WindowSize{Cols: 80})
	s := session.New("s-1", conn, logger, fixedClock(time.Unix(1_700_000_000, 0)))
	_ = menu.Demo("").Run(s)
	s.Close()
	logger.Close()

	out := conn.output()
	if bytes.Contains([]byte(out), []byte{0x1b}) {
		t.Fatalf("B&W session leaked escape codes:\n%q", out)
	}
	if !strings.Contains(out, "Main Menu") || !strings.Contains(out, "NO CARRIER") {
		t.Fatalf("B&W session missing expected text:\n%s", out)
	}
}

func hasActivity(evs []audit.Event, action string) bool {
	for _, e := range evs {
		if e.Type == audit.TypeActivity && e.Action == action {
			return true
		}
	}
	return false
}
