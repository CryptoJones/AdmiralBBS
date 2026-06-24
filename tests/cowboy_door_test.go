package tests

import (
	"testing"

	"admiralbbs/src/store"
)

// The BBS registers Console Cowboy 2026 as a resident door, idempotently, and
// can move its address without creating duplicates.
func TestEnsureResidentDoorIdempotent(t *testing.T) {
	s, _ := openTestStore(t)
	doors := s.Doors()

	if err := doors.EnsureResidentDoor("Console Cowboy 2026", "tcp", "127.0.0.1:4000", 0); err != nil {
		t.Fatal(err)
	}
	// Re-register at a new address — should update, not duplicate.
	if err := doors.EnsureResidentDoor("Console Cowboy 2026", "tcp", "10.0.0.5:4000", 0); err != nil {
		t.Fatal(err)
	}

	n, _ := doors.Count()
	if n != 1 {
		t.Fatalf("door count = %d, want 1 (no duplicate)", n)
	}
	visible, _ := doors.Visible(0)
	var found *store.Door
	for _, d := range visible {
		if d.Name == "Console Cowboy 2026" {
			found = d
		}
	}
	if found == nil {
		t.Fatal("Console Cowboy 2026 not visible")
	}
	if found.Kind != store.KindResident || found.Address != "10.0.0.5:4000" {
		t.Fatalf("door not updated to resident@new addr: %+v", found)
	}
}
