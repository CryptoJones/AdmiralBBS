package tests

import (
	"strings"
	"testing"

	"admiralbbs/src/game/cowboy"
)

// routeToNet walks a player from the start room up into the Grid Node.
func routeToNet(w *cowboy.World, p *cowboy.Player) {
	w.Command(p, "out")  // capsule pod -> neon_alley (the street)
	w.Command(p, "east") // the_sprawl
	w.Command(p, "east") // corpo_plaza
	w.Command(p, "east") // data_port
	w.Command(p, "up")   // the_net
}

func TestCowboyRAMBreachEconomy(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	out, buf := sink()
	p := w.Connect("Case", out)
	routeToNet(w, p)
	if p.RoomID != "the_net" {
		t.Fatalf("expected the_net, at %s", p.RoomID)
	}
	// Make this a survivable, controlled breach test.
	p.MaxHP, p.HP = 1000, 1000
	p.RAM = 2 // only two full-power breaches before it sputters

	w.Command(p, "attack ice")
	w.Tick() // breach 1: RAM 2 -> 1
	w.Tick() // breach 2: RAM 1 -> 0
	if p.RAM != 0 {
		t.Fatalf("RAM after two breaches = %d, want 0", p.RAM)
	}
	w.Tick() // breach 3: no RAM -> sputters
	if !strings.Contains(buf.String(), "Low RAM") {
		t.Error("expected a low-RAM sputter once RAM hit 0")
	}

	// Out of combat, RAM regenerates (verify on a fresh, idle player in meatspace).
	out2, _ := sink()
	q := w.Connect("Idle", out2) // neon_alley, not fighting
	q.RAM = 0
	w.Tick()
	if q.RAM <= 0 {
		t.Fatalf("RAM should regenerate out of combat, got %d", q.RAM)
	}
}

func TestCowboyMultiStageICE(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	out, buf := sink()
	p := w.Connect("Case", out)
	routeToNet(w, p)
	w.Command(p, "down") // ice_wall (Sentinel Lattice — the Gauntlet ICE)
	if p.RoomID != "ice_wall" {
		t.Fatalf("expected ice_wall, at %s", p.RoomID)
	}
	// Buff so we survive the gauntlet and have RAM to spare.
	p.Intelligence, p.MaxHP, p.HP, p.RAM = 40, 2000, 2000, 200

	// Clear the whole gauntlet. The intermediate stages morph (no reward); only
	// the final lethal lock pays out — 700 XP, which also levels the player up.
	w.Command(p, "attack gauntlet")
	for i := 0; i < 80 && p.Level == 1; i++ {
		w.Command(p, "attack gauntlet") // re-target the morphed form each round
		w.Tick()
	}
	s := buf.String()
	if strings.Count(s, "reconfigures") < 2 {
		t.Errorf("multi-stage ICE should morph at least twice; output:\n%s", lastLines(s))
	}
	if !strings.Contains(s, "lethal lock") || !strings.Contains(s, "destroyed") {
		t.Error("the final gauntlet stage should be destroyed")
	}
	if p.Level < 2 {
		t.Fatalf("clearing the gauntlet (700 XP) should have leveled the player up, level=%d", p.Level)
	}
}

// After the multi-stage Gauntlet ICE is fully beaten, it must respawn back in
// the Sentinel Lattice as its FIRST form — not vanish into the void (the bug:
// the morphed template had no home room).
func TestCowboyGauntletRespawnsAsFirstForm(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	out, buf := sink()
	p := w.Connect("Case", out)
	routeToNet(w, p)
	w.Command(p, "down") // ice_wall
	p.Intelligence, p.MaxHP, p.RAM = 60, 100000, 100000
	p.HP = p.MaxHP

	// Beat the whole gauntlet (final stage awards 700 XP -> a level-up).
	w.Command(p, "attack gauntlet")
	for i := 0; i < 120 && p.Level == 1; i++ {
		w.Command(p, "attack gauntlet")
		w.Tick()
	}
	if p.Level == 1 {
		t.Fatal("never finished the gauntlet")
	}
	mark := buf.Len()

	// Tick past the respawn cooldown; the player is still in the Lattice, so the
	// reinitialize broadcast (in ice_wall) reaches them with the FIRST-form name.
	for i := 0; i < 30; i++ {
		w.Tick()
	}
	after := buf.String()[mark:]
	if !strings.Contains(after, "white shell") {
		t.Fatalf("gauntlet should respawn as its first form in the Lattice; post-kill output:\n%s", lastLines(after))
	}
}

func TestCowboyPvPDuel(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	o1, b1 := sink()
	p1 := w.Connect("Case", o1)
	o2, b2 := sink()
	p2 := w.Connect("Molly", o2)
	routeToNet(w, p1)
	routeToNet(w, p2)
	if p1.RoomID != "the_net" || p2.RoomID != "the_net" {
		t.Fatalf("both should be in the_net: %s / %s", p1.RoomID, p2.RoomID)
	}

	p1.Intelligence, p1.RAM = 40, 50 // strong attacker
	p1Eddies := p1.Eddies
	p2.Eddies = 100 // something to siphon
	// Low enough that p1's breach (~24) finishes it in one round, but high enough
	// that an aggro'd White ICE hit (~7) can't kill it first — so the kill is the duel's.
	p2.HP = 20

	w.Command(p1, "attack molly")
	if p1.RoomID != "the_net" || !strings.Contains(b1.String(), "netrun duel") {
		t.Fatalf("PvP should have engaged in the Net; out:\n%s", b1.String())
	}
	w.Tick() // p1 breaches p2 -> flatline

	if !strings.Contains(b2.String(), "DECK IS FRIED") {
		t.Errorf("loser should be flatlined; out:\n%s", lastLines(b2.String()))
	}
	if p2.RoomID != "capsule" {
		t.Errorf("loser should respawn at the start capsule, at %s", p2.RoomID)
	}
	if p1.Eddies <= p1Eddies {
		t.Errorf("winner should siphon eddies: before %d after %d", p1Eddies, p1.Eddies)
	}
}

// In the safe zone (the street outside the clone pods), drawing on another
// runner gets the AGGRESSOR flatlined by a security drone; the target is untouched.
func TestCowboySafeZoneSecurityKill(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	o1, _ := sink()
	p1 := w.Connect("Case", o1)
	w.Command(p1, "out")
	o2, _ := sink()
	p2 := w.Connect("Molly", o2)
	w.Command(p2, "out") // both in neon_alley (no-violence zone)

	hp2 := p2.HP
	w.Command(p1, "attack molly")
	if p1.RoomID != "capsule" {
		t.Fatalf("aggressor should be security-killed and re-sleeved; at %s", p1.RoomID)
	}
	if p2.HP != hp2 {
		t.Fatalf("the target should be unharmed: %d -> %d", hp2, p2.HP)
	}
}

// PvP is live in the REST of meatspace: attacking a runner starts a duel.
func TestCowboyPvPInMeatspace(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	o1, b1 := sink()
	p1 := w.Connect("Case", o1)
	o2, _ := sink()
	p2 := w.Connect("Molly", o2)
	for _, p := range []*cowboy.Player{p1, p2} {
		w.Command(p, "out")
		w.Command(p, "east") // the_sprawl (not a safe zone)
	}
	if p1.RoomID != "the_sprawl" || p2.RoomID != "the_sprawl" {
		t.Fatalf("both should be in the_sprawl: %s/%s", p1.RoomID, p2.RoomID)
	}
	w.Command(p1, "attack molly")
	if !strings.Contains(b1.String(), "duel") {
		t.Fatalf("attacking a runner in open meatspace should start a duel; got:\n%s", b1.String())
	}
}

func TestCowboyDeckPersistsRAM(t *testing.T) {
	store := cowboy.NewMemStore()
	w := cowboy.NewWorld(store)
	out, _ := sink()
	p := w.Connect("Case", out)
	baseMax := 5 + p.Intelligence/2

	w.Command(p, "out")   // re-sleeve bay -> neon_alley
	w.Command(p, "south") // chrome_bar vendor
	p.Eddies = 500
	w.Command(p, "buy cyberdeck")
	if p.DeckBonus != 8 {
		t.Fatalf("cyberdeck not installed: DeckBonus=%d", p.DeckBonus)
	}
	w.Disconnect(p)

	w2 := cowboy.NewWorld(store)
	out2, _ := sink()
	p2 := w2.Connect("Case", out2)
	if p2.DeckBonus != 8 {
		t.Fatalf("deck bonus not persisted: %d", p2.DeckBonus)
	}
	// Max RAM should reflect the persisted deck (8 over the stock base).
	if got := 5 + p2.Intelligence/2 + p2.DeckBonus; got != baseMax+8 {
		t.Fatalf("max RAM = %d, want %d", got, baseMax+8)
	}
}
