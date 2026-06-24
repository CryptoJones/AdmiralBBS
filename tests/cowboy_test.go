package tests

import (
	"strings"
	"testing"

	"admiralbbs/src/game/cowboy"
)

// sink captures a player's output for assertions.
func sink() (func(string), *strings.Builder) {
	var b strings.Builder
	return func(s string) { b.WriteString(s) }, &b
}

// alwaysHit makes combat deterministic: roll(n) returns n-1 (max), so to-hit
// always succeeds and flee always fails.
func alwaysHit(n int) int { return n - 1 }

func TestCowboyConnectAndLook(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	out, buf := sink()
	p := w.Connect("Case", out)
	if p.RoomID != "neon_alley" || p.Level != 1 || p.HP <= 0 {
		t.Fatalf("new character wrong: %+v", p)
	}
	s := buf.String()
	for _, want := range []string{"You jack in as Case", "Neon Alley", "Exits:"} {
		if !strings.Contains(s, want) {
			t.Errorf("connect output missing %q", want)
		}
	}
}

func TestCowboyMovement(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	out, buf := sink()
	p := w.Connect("Case", out)
	w.Command(p, "east")
	if p.RoomID != "the_sprawl" {
		t.Fatalf("east -> %s, want the_sprawl", p.RoomID)
	}
	w.Command(p, "north")
	if p.RoomID != "back_alley" {
		t.Fatalf("north -> %s, want back_alley", p.RoomID)
	}
	if !strings.Contains(buf.String(), "The Sprawl") {
		t.Error("movement didn't show the destination room")
	}
}

func TestCowboyCombatKillAndReward(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	out, buf := sink()
	p := w.Connect("Case", out)
	w.Command(p, "east")  // the_sprawl
	w.Command(p, "north") // back_alley (street ganger)
	w.Command(p, "attack ganger")
	for i := 0; i < 8 && p.XP == 0; i++ {
		w.Tick()
	}
	if p.XP != 25 {
		t.Fatalf("XP after killing ganger = %d, want 25", p.XP)
	}
	if p.Eddies != 60 { // 50 start + 10 bounty
		t.Fatalf("eddies = %d, want 60", p.Eddies)
	}
	if !strings.Contains(buf.String(), "destroyed") {
		t.Error("kill message missing")
	}
	if p.HP <= 0 {
		t.Error("player should have survived a lone ganger")
	}
}

func TestCowboyMultiplayerVisibilityAndChat(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	o1, b1 := sink()
	p1 := w.Connect("Case", o1)
	o2, b2 := sink()
	w.Connect("Molly", o2) // both start in neon_alley
	if !strings.Contains(b1.String(), "Molly") {
		t.Error("Case should see Molly materialize")
	}
	w.Command(p1, "say jack in, choom")
	// (ANSI color resets sit between the speaker and the message, so check the
	// two fragments rather than one contiguous string.)
	if !strings.Contains(b2.String(), "Case says:") || !strings.Contains(b2.String(), "jack in, choom") {
		t.Error("Molly should hear Case say")
	}
	o3, b3 := sink()
	p3 := w.Connect("Watcher", o3)
	w.Command(p3, "who")
	s := b3.String()
	if !strings.Contains(s, "Case") || !strings.Contains(s, "Molly") || !strings.Contains(s, "Watcher") {
		t.Errorf("who should list all three; got:\n%s", s)
	}
}

func TestCowboyShop(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	out, buf := sink()
	p := w.Connect("Case", out)
	w.Command(p, "south") // chrome_bar (vendor)
	w.Command(p, "list")
	if !strings.Contains(buf.String(), "stimpak") {
		t.Error("vendor list should show wares")
	}

	// Can't afford the blade at 50 eddies.
	w.Command(p, "buy ice-breaker")
	if p.WeaponBonus != 0 {
		t.Fatal("bought a weapon without enough eddies")
	}
	// Stipend, then buy and equip.
	p.Eddies = 500
	w.Command(p, "buy ice-breaker")
	if p.WeaponBonus != 5 || p.WeaponName != "ice-breaker" {
		t.Fatalf("weapon not equipped: bonus=%d name=%q", p.WeaponBonus, p.WeaponName)
	}
	// Stimpak heals.
	w.Command(p, "buy stimpak")
	p.HP = 1
	w.Command(p, "use stimpak")
	if p.HP <= 1 {
		t.Fatalf("stimpak didn't heal: HP=%d", p.HP)
	}
}

func TestCowboyNetBreachVerb(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	out, buf := sink()
	p := w.Connect("Case", out)
	// Route into the Net: sprawl -> corpo_plaza -> data_port -> up.
	w.Command(p, "east")
	w.Command(p, "east")
	w.Command(p, "east")
	w.Command(p, "up")
	if p.RoomID != "the_net" {
		t.Fatalf("expected to reach the_net, at %s", p.RoomID)
	}
	w.Command(p, "attack ice")
	if !strings.Contains(buf.String(), "breach protocol") {
		t.Error("attacking in the Net should be a breach, not a melee strike")
	}
}

func TestCowboyPersistence(t *testing.T) {
	store := cowboy.NewMemStore()

	w1 := cowboy.NewWorld(store)
	out, _ := sink()
	p := w1.Connect("Case", out)
	p.Eddies = 999
	p.XP = 42
	w1.Disconnect(p)

	w2 := cowboy.NewWorld(store)
	out2, _ := sink()
	p2 := w2.Connect("Case", out2)
	if p2.Eddies != 999 || p2.XP != 42 {
		t.Fatalf("progress not persisted: eddies=%d xp=%d", p2.Eddies, p2.XP)
	}
}

func TestCowboyOneSessionPerName(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	out, _ := sink()
	w.Connect("Case", out)
	if !w.Online("case") {
		t.Error("Online should be case-insensitive")
	}
}
