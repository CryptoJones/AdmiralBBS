package tests

import (
	"strings"
	"testing"

	"admiralbbs/src/game/cowboy"
)

// killGangersInAlley drives auto-combat in the back alley until the player has
// scored n ganger kills toward the clear_alley bounty (aggressive gangers
// re-engage on respawn, so ticking is enough). Assumes p is in back_alley.
func killGangersInAlley(t *testing.T, w *cowboy.World, p *cowboy.Player, n int) {
	t.Helper()
	for i := 0; i < 600 && p.Quests["clear_alley"] < n; i++ {
		w.Command(p, "attack ganger")
		w.Tick()
		if p.HP <= 0 {
			t.Fatalf("player unexpectedly died at kill %d", p.Quests["clear_alley"])
		}
	}
}

func TestCowboyQuestAcceptKillClaim(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	out, buf := sink()
	p := w.Connect("Case", out)

	// Accept "Clear the Back Alley" (3 gangers) at the fixer off the street.
	w.Command(p, "out")   // re-sleeve bay -> neon_alley
	w.Command(p, "south") // chrome_bar (fixer)
	w.Command(p, "quests")
	if !strings.Contains(buf.String(), "Clear the Back Alley") {
		t.Fatal("bounty board should list the alley job")
	}
	w.Command(p, "accept 1")
	if _, ok := p.Quests["clear_alley"]; !ok {
		t.Fatal("accept didn't add the bounty")
	}

	startXP, startEddies := p.XP, p.Eddies

	// To the back alley and clear it.
	w.Command(p, "north") // neon_alley
	w.Command(p, "east")  // the_sprawl
	w.Command(p, "north") // back_alley
	killGangersInAlley(t, w, p, 3)
	if p.Quests["clear_alley"] < 3 {
		t.Fatalf("quest progress = %d, want 3", p.Quests["clear_alley"])
	}

	// Claiming away from a fixer must fail.
	w.Command(p, "claim")
	if _, still := p.Quests["clear_alley"]; !still {
		t.Fatal("claim should not pay out away from a fixer")
	}
	// Back to a fixer and claim.
	w.Command(p, "south") // the_sprawl
	w.Command(p, "west")  // neon_alley
	w.Command(p, "south") // chrome_bar (fixer)
	w.Command(p, "claim")
	if _, still := p.Quests["clear_alley"]; still {
		t.Fatal("claim at a fixer should clear the bounty")
	}
	if p.XP <= startXP || p.Eddies <= startEddies {
		t.Fatalf("claim paid nothing: dXP=%d dEddies=%d", p.XP-startXP, p.Eddies-startEddies)
	}
}

func TestCowboyMinLevelGate(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	out, buf := sink()
	p := w.Connect("Rookie", out) // level 1
	w.Command(p, "out")           // -> neon_alley
	w.Command(p, "south")         // fixer
	w.Command(p, "accept 4")      // "Ghost in the Machine" needs level 5
	if _, ok := p.Quests["ghost_machine"]; ok {
		t.Fatal("a level-1 rookie should be blocked from the level-5 bounty")
	}
	if !strings.Contains(buf.String(), "need level") {
		t.Error("expected a level-requirement message")
	}
}

func TestCowboyLevelCap(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	out, _ := sink()
	p := w.Connect("Maxed", out)

	// Pile on far more XP than the whole curve needs, then trigger one kill so
	// checkLevelUp runs. It must stop exactly at the cap and zero excess XP.
	w.Command(p, "out")   // -> neon_alley
	w.Command(p, "east")  // the_sprawl
	w.Command(p, "north") // back_alley
	p.XP = 100_000_000
	for i := 0; i < 30 && p.Level < cowboy.MaxLevel; i++ {
		w.Command(p, "attack ganger")
		w.Tick()
	}
	if p.Level != cowboy.MaxLevel {
		t.Fatalf("level = %d, want exactly the cap %d", p.Level, cowboy.MaxLevel)
	}
	if p.XP != 0 {
		t.Fatalf("XP at cap = %d, want 0 (excess discarded)", p.XP)
	}
}
