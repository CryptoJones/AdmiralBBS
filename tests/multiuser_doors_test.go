package tests

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"admiralbbs/src/doors"
)

// A door's per-node working dir PERSISTS across plays, and the shared dir
// ($DOORSHARE) is where multiplayer state lives — proving doors aren't
// wiped-every-time single-player-only.
func TestDoorStatePersistsAndShares(t *testing.T) {
	dir := t.TempDir()
	work := filepath.Join(dir, "node1")
	share := filepath.Join(dir, "shared")
	script := `echo x >> state; echo y >> "$DOORSHARE/world"`

	for i := 0; i < 2; i++ {
		err := doors.Launch(
			pipeRW{r: strings.NewReader(""), w: io.Discard},
			"/bin/sh", []string{"-c", script},
			doors.DropInfo{Handle: "tester"},
			doors.Opts{Timeout: 5 * time.Second, WorkDir: work, ShareDir: share},
		)
		if err != nil {
			t.Fatalf("launch %d: %v", i, err)
		}
	}

	state, err := os.ReadFile(filepath.Join(work, "state"))
	if err != nil || strings.Count(string(state), "x") != 2 {
		t.Fatalf("per-node state not persisted across plays: %q err=%v", state, err)
	}
	world, err := os.ReadFile(filepath.Join(share, "world"))
	if err != nil || strings.Count(string(world), "y") != 2 {
		t.Fatalf("shared multiplayer dir not working: %q err=%v", world, err)
	}
}
