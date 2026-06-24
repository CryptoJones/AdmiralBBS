package tests

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"admiralbbs/src/doors"
)

// rw wires a fixed input reader to a captured output buffer (a fake session).
type rw struct {
	in  io.Reader
	out *bytes.Buffer
}

func (x *rw) Read(p []byte) (int, error)  { return x.in.Read(p) }
func (x *rw) Write(p []byte) (int, error) { return x.out.Write(p) }

func newRW(input string) *rw { return &rw{in: strings.NewReader(input), out: &bytes.Buffer{}} }

// SEC-1: a door must NOT inherit the daemon's environment — the master key and
// every secret must be invisible.
func TestDoorEnvIsScrubbed(t *testing.T) {
	os.Setenv("ADMIRALBBS_KEY", "supersecret-master-key")
	defer os.Unsetenv("ADMIRALBBS_KEY")

	conn := newRW("")
	// The "door" prints its env + reads the dropfile handle.
	script := `printenv ADMIRALBBS_KEY || echo NOKEY; echo "HANDLE=$(sed -n 7p "$DOORFILE" | tr -d '\r')"`
	err := doors.Launch(conn, "/bin/sh", []string{"-c", script},
		doors.DropInfo{BBSName: "AdmiralBBS", Handle: "zerocool", AccessLevel: 50, ANSI: true},
		doors.Opts{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	out := conn.out.String()
	if strings.Contains(out, "supersecret-master-key") {
		t.Fatalf("MASTER KEY LEAKED into the door env:\n%s", out)
	}
	if !strings.Contains(out, "NOKEY") {
		t.Fatalf("expected NOKEY, got:\n%s", out)
	}
	if !strings.Contains(out, "HANDLE=zerocool") {
		t.Fatalf("dropfile handle not readable by door:\n%s", out)
	}
}

// A runaway door is killed at the wall-clock timeout (its whole process group).
func TestDoorTimeoutKills(t *testing.T) {
	conn := newRW("")
	start := time.Now()
	_ = doors.Launch(conn, "/bin/sh", []string{"-c", "sleep 30"}, doors.DropInfo{Handle: "x"},
		doors.Opts{Timeout: 1 * time.Second})
	if elapsed := time.Since(start); elapsed > 8*time.Second {
		t.Fatalf("runaway door not killed promptly: took %v", elapsed)
	}
}

func TestDoor32Dropfile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "door32.sys")
	if err := doors.WriteDoor32(p, doors.DropInfo{BBSName: "AdmiralBBS", Handle: "zerocool", AccessLevel: 100, MinutesLeft: 30, Node: 1, ANSI: true}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	lines := strings.Split(strings.ReplaceAll(strings.TrimRight(string(data), "\r\n"), "\r\n", "\n"), "\n")
	if len(lines) != 11 {
		t.Fatalf("door32.sys has %d lines, want 11", len(lines))
	}
	if lines[6] != "zerocool" || lines[7] != "100" || lines[9] != "1" {
		t.Fatalf("dropfile fields wrong: handle=%q access=%q emu=%q", lines[6], lines[7], lines[9])
	}
}

func TestDoorsRepoAccessGating(t *testing.T) {
	s, _ := openTestStore(t)
	if _, err := s.Doors().Create("Number Guess", "doors/numguess.sh", "door32.sys", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Doors().Create("SysOp Diagnostics", "diag", "door32.sys", 100); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Doors().Visible(50); len(got) != 1 {
		t.Fatalf("member should see 1 door, saw %d", len(got))
	}
	if got, _ := s.Doors().Visible(100); len(got) != 2 {
		t.Fatalf("sysop should see 2 doors, saw %d", len(got))
	}
}
