package tests

import (
	"strings"
	"testing"

	"admiralbbs/src/game/cowboy"
)

func routeToDeepNet(w *cowboy.World, p *cowboy.Player) {
	routeToNet(w, p)     // -> the_net
	w.Command(p, "down") // ice_wall
	w.Command(p, "down") // deep_net
}

func TestCowboyProgramsRAMAndGates(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	out, buf := sink()
	p := w.Connect("Case", out)
	routeToNet(w, p)
	p.Intelligence, p.MaxHP, p.HP, p.RAM = 40, 2000, 2000, 20

	w.Command(p, "attack ice") // engage White ICE
	ramBefore := p.RAM
	w.Command(p, "run hammer") // costs 4 RAM
	if p.RAM != ramBefore-4 {
		t.Fatalf("hammer RAM cost: before %d after %d, want -4", ramBefore, p.RAM)
	}
	if !strings.Contains(buf.String(), "execute Hammer") {
		t.Error("hammer should report execution")
	}

	// Medic repairs HP.
	p.HP = 10
	w.Command(p, "run medic")
	if p.HP <= 10 {
		t.Fatalf("medic didn't heal: HP=%d", p.HP)
	}

	// RAM-gated: too little RAM is refused without spending.
	p.RAM = 1
	w.Command(p, "run hammer")
	if p.RAM != 1 || !strings.Contains(buf.String(), "Not enough RAM") {
		t.Errorf("low-RAM hammer should be refused; RAM=%d", p.RAM)
	}

	// Net-only: a fresh runner in meatspace can't run a Net program.
	o2, b2 := sink()
	q := w.Connect("Rookie", o2)
	w.Command(q, "run scalpel")
	if !strings.Contains(b2.String(), "only runs inside the Net") {
		t.Error("net-only program should be refused in meatspace")
	}
}

func TestCowboyMirrorShieldReducesDamage(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	out, _ := sink()
	p := w.Connect("Case", out)
	routeToDeepNet(w, p)
	if p.RoomID != "deep_net" {
		t.Fatalf("expected deep_net, at %s", p.RoomID)
	}
	p.Intelligence, p.MaxHP, p.RAM = 40, 4000, 80
	p.HP = p.MaxHP
	w.Command(p, "attack black") // engage Black ICE (survives several rounds)

	// One round with no shield.
	hp0 := p.HP
	w.Tick()
	lossNoShield := hp0 - p.HP

	// Mirror up, one round with shield.
	p.HP = p.MaxHP
	w.Command(p, "run mirror")
	hp1 := p.HP
	w.Tick()
	lossShield := hp1 - p.HP

	if !(lossShield < lossNoShield) {
		t.Fatalf("Mirror should reduce damage: noShield=%d shield=%d", lossNoShield, lossShield)
	}
}

func TestCowboyPartyXPShareAndChat(t *testing.T) {
	// `party` is engine-internal, so verify crews behaviorally: the join
	// notice, shared kill XP, crew radio, and dissolution on leave.
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	o1, b1 := sink()
	p1 := w.Connect("Case", o1)
	o2, b2 := sink()
	p2 := w.Connect("Molly", o2)

	w.Command(p1, "group Molly")
	if !strings.Contains(b1.String(), "joins the crew") {
		t.Fatal("group should form a crew (no join notice)")
	}

	// Both to the back alley.
	for _, p := range []*cowboy.Player{p1, p2} {
		w.Command(p, "east")
		w.Command(p, "north")
	}
	if p1.RoomID != "back_alley" || p2.RoomID != "back_alley" {
		t.Fatalf("crew should both be in back_alley: %s/%s", p1.RoomID, p2.RoomID)
	}

	// p1 lands the kill; p2 (in the same room) shares the XP.
	w.Command(p1, "attack ganger")
	for i := 0; i < 8 && p1.XP == 0; i++ {
		w.Command(p1, "attack ganger")
		w.Tick()
	}
	if p1.XP == 0 || p2.XP == 0 {
		t.Fatalf("crew should share kill XP: p1=%d p2=%d", p1.XP, p2.XP)
	}

	// Crew radio reaches the other member, wherever they are.
	w.Command(p1, "gsay form up")
	if !strings.Contains(b2.String(), "[crew]") || !strings.Contains(b2.String(), "form up") {
		t.Error("gsay should reach crewmates")
	}

	// Leaving a 2-person crew dissolves it: p1 can no longer radio a crew.
	w.Command(p2, "leave")
	w.Command(p1, "gsay anyone there")
	if !strings.Contains(b1.String(), "no crew") {
		t.Error("crew should dissolve when it drops below two")
	}
}

func TestCowboyLeaderboard(t *testing.T) {
	store := cowboy.NewMemStore()
	w := cowboy.NewWorld(store)

	mk := func(name string, level, xp int) {
		out, _ := sink()
		p := w.Connect(name, out)
		p.Level, p.XP = level, xp
		w.Disconnect(p) // persists
	}
	mk("Ace", 10, 50)
	mk("Boo", 5, 999)
	mk("Cid", 10, 200)

	top, err := store.Top(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 3 || top[0].Name != "Cid" || top[1].Name != "Ace" || top[2].Name != "Boo" {
		t.Fatalf("ranking wrong: %+v", top)
	}

	// The leaderboard command renders the ranked names.
	w2 := cowboy.NewWorld(store)
	out, buf := sink()
	v := w2.Connect("Viewer", out)
	w2.Command(v, "leaderboard")
	s := buf.String()
	if !strings.Contains(s, "TOP CONSOLE COWBOYS") || !strings.Contains(s, "Cid") || !strings.Contains(s, "Ace") {
		t.Errorf("leaderboard output missing names:\n%s", lastLines(s))
	}
}
