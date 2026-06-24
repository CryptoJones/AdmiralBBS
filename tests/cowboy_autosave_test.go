package tests

import (
	"testing"

	"admiralbbs/src/game/cowboy"
)

// SaveAll persists connected players' progress WITHOUT a clean disconnect — the
// basis for periodic autosave + save-on-shutdown, so a server restart/crash
// doesn't lose progress since login.
func TestCowboySaveAllPersistsConnectedPlayers(t *testing.T) {
	store := cowboy.NewMemStore()
	w1 := cowboy.NewWorld(store)
	out, _ := sink()
	p := w1.Connect("Case", out)
	p.XP = 99
	p.Eddies = 777 // gained mid-session; no Disconnect called

	w1.SaveAll()

	// A fresh world over the same store reloads the saved progress.
	w2 := cowboy.NewWorld(store)
	out2, _ := sink()
	p2 := w2.Connect("Case", out2)
	if p2.XP != 99 || p2.Eddies != 777 {
		t.Fatalf("SaveAll didn't persist mid-session progress: XP=%d Eddies=%d", p2.XP, p2.Eddies)
	}
}
