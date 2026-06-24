package tests

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

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
	if p.RoomID != "capsule" || p.Level != 1 || p.HP <= 0 {
		t.Fatalf("new character wrong: %+v", p)
	}
	s := buf.String()
	for _, want := range []string{"You jack in as Case", "Re-Sleeve Bay", "Exits:"} {
		if !strings.Contains(s, want) {
			t.Errorf("connect output missing %q", want)
		}
	}
}

func TestCowboyMovement(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	out, buf := sink()
	p := w.Connect("Case", out)
	w.Command(p, "out") // capsule -> neon_alley (the street)
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
	w.Command(p, "out")   // capsule -> neon_alley
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
	w.Command(p1, "out") // to the street (capsule is private)
	o2, b2 := sink()
	p2 := w.Connect("Molly", o2)
	w.Command(p2, "out") // Molly arrives in the street where Case is
	if !strings.Contains(b1.String(), "Molly") {
		t.Error("Case should see Molly arrive in the street")
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
	w.Command(p, "out")   // capsule -> neon_alley
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
	// Route into the Net: street -> sprawl -> corpo_plaza -> data_port -> up.
	w.Command(p, "out")
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

// Inventory shows the player's credits (eddies), not just items — players expect
// their money on the same screen as their carried goods.
func TestCowboyInventoryShowsCredits(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	out, b := sink()
	p := w.Connect("Case", out)
	p.Eddies = 1234
	b.Reset()
	w.Command(p, "inventory")
	if !strings.Contains(b.String(), "1234") {
		t.Fatalf("inventory did not show credits; got:\n%s", b.String())
	}
}

// Using a single-use heal item at full HP must NOT consume it (no silent waste).
func TestCowboyUseStimpakAtFullHPDoesNotWaste(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	out, b := sink()
	p := w.Connect("Case", out) // starts with 1 stimpak, full HP
	p.HP = p.MaxHP
	b.Reset()
	w.Command(p, "use stimpak")
	if p.Inv["stimpak"] != 1 {
		t.Fatalf("stimpak wasted at full HP; remaining=%d, want 1", p.Inv["stimpak"])
	}
	if !strings.Contains(b.String(), "already full") {
		t.Fatalf("expected full-HP refusal; got:\n%s", b.String())
	}
}

// A hurt player CAN use the stimpak: it heals and is consumed.
func TestCowboyUseStimpakWhenHurtHeals(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	out, _ := sink()
	p := w.Connect("Case", out)
	p.HP = 1
	w.Command(p, "use stimpak")
	if p.Inv["stimpak"] != 0 {
		t.Fatalf("stimpak not consumed when hurt; remaining=%d", p.Inv["stimpak"])
	}
	if p.HP <= 1 {
		t.Fatalf("stimpak did not heal; HP=%d", p.HP)
	}
}

// `use` with no argument asks what, instead of "You don't have a ."
func TestCowboyUseEmptyArg(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	out, b := sink()
	p := w.Connect("Case", out)
	b.Reset()
	w.Command(p, "use")
	if !strings.Contains(b.String(), "Use what?") {
		t.Fatalf("expected 'Use what?'; got:\n%s", b.String())
	}
}

// The welcome banner box must render with all rows the same visible width, so
// the ╔/║/╝ borders stay vertically aligned (regression: hand-spaced padding
// drifted, and the em-dash in the tagline threw off byte-based counts).
func TestCowboyBannerBoxAligned(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	var b strings.Builder
	w.Connect("Case", func(s string) { b.WriteString(s) }) // enter() sends the banner
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	var widths []int
	for _, ln := range strings.Split(b.String(), "\n") {
		clean := ansi.ReplaceAllString(strings.TrimRight(ln, "\r"), "")
		if strings.HasPrefix(clean, "╔") || strings.HasPrefix(clean, "║") || strings.HasPrefix(clean, "╚") {
			widths = append(widths, utf8.RuneCountInString(clean))
		}
	}
	if len(widths) != 4 {
		t.Fatalf("expected 4 banner box rows, found %d", len(widths))
	}
	for i, wd := range widths {
		if wd != widths[0] {
			t.Fatalf("banner row %d width=%d != top-border width=%d — box misaligned", i, wd, widths[0])
		}
	}
}

// New runners re-sleeve in a PRIVATE bay (spawn-safe + isolated) and step OUT
// into the street — so a respawn can't be spawn-camped.
func TestCowboySpawnBayIsPrivate(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	o1, b1 := sink()
	p1 := w.Connect("Case", o1)
	o2, _ := sink()
	p2 := w.Connect("Molly", o2)
	if p1.RoomID != "capsule" || p2.RoomID != "capsule" {
		t.Fatalf("new runners should spawn in the re-sleeve bay: %s/%s", p1.RoomID, p2.RoomID)
	}
	b1.Reset()
	w.Command(p1, "look")
	if strings.Contains(b1.String(), "Molly") {
		t.Fatal("the re-sleeve bay must be private — no one else is visible in it")
	}
	w.Command(p1, "out")
	if p1.RoomID != "neon_alley" {
		t.Fatalf("OUT should lead to the street; at %s", p1.RoomID)
	}
}

// When a crew leader flatlines, leadership passes to the longest-tenured survivor
// (a dead runner doesn't keep leading), and the leader re-sleeves at full HP.
func TestCowboyLeaderDeathPassesLeadership(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	o1, _ := sink()
	leader := w.Connect("Case", o1) // joins first -> leader
	o2, b2 := sink()
	member := w.Connect("Molly", o2)
	w.Command(leader, "group Molly") // invite
	w.Command(member, "accept")      // consent -> Case leads [Case, Molly]

	w.Command(leader, "out")
	w.Command(leader, "east")
	w.Command(leader, "north") // back_alley (ganger)
	leader.HP = 1
	w.Command(leader, "attack ganger")
	b2.Reset()
	for i := 0; i < 6 && leader.RoomID != "capsule"; i++ {
		w.Tick()
	}
	if leader.RoomID != "capsule" {
		t.Fatalf("leader should have flatlined and re-sleeved; at %s hp=%d", leader.RoomID, leader.HP)
	}
	if leader.HP != leader.MaxHP {
		t.Fatalf("fresh clone should be full HP: %d/%d", leader.HP, leader.MaxHP)
	}
	if !strings.Contains(b2.String(), "now leads the crew") {
		t.Fatalf("leadership should pass to the surviving member; got:\n%s", b2.String())
	}
}

// Altered-Carbon death loop: dying drops your old sleeve (items + cyberware) and
// the clone wakes stripped; another runner can loot it, re-install the cyberware
// at a ripperdoc, and give recovered gear back.
func TestCowboyCorpseLootInstallGive(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	o1, _ := sink()
	victim := w.Connect("Case", o1)
	o2, b2 := sink()
	helper := w.Connect("Molly", o2)

	// Victim kits up: weapon + cyberdeck + stimpaks.
	victim.WeaponName, victim.WeaponBonus = "ice-breaker", 5
	victim.DeckBonus = 8
	victim.Inv["stimpak"] = 2

	// Both into the back alley; the ganger flatlines the victim (HP 1).
	for _, p := range []*cowboy.Player{victim, helper} {
		w.Command(p, "out")
		w.Command(p, "east")
		w.Command(p, "north")
	}
	victim.HP = 1
	w.Command(victim, "attack ganger")
	for i := 0; i < 6 && victim.RoomID != "capsule"; i++ {
		w.Tick()
	}
	if victim.RoomID != "capsule" {
		t.Fatalf("victim should have re-sleeved; at %s", victim.RoomID)
	}
	// The fresh clone wakes stripped of gear AND cyberware.
	if victim.WeaponBonus != 0 || victim.DeckBonus != 0 || len(victim.Inv) != 0 {
		t.Fatalf("clone should wake stripped: wpn=%d deck=%d inv=%v", victim.WeaponBonus, victim.DeckBonus, victim.Inv)
	}

	// Helper (still in the alley) loots the sleeve.
	b2.Reset()
	w.Command(helper, "loot")
	if helper.Inv["stimpak"] != 3 || helper.Inv["ice-breaker"] != 1 || helper.Inv["cyberdeck"] != 1 {
		t.Fatalf("loot should recover gear+cyberware (helper had 1 stimpak): %v", helper.Inv)
	}
	if !strings.Contains(b2.String(), "Salvaged cyberware") {
		t.Fatal("loot should flag salvaged cyberware")
	}

	// Install needs a ripperdoc — refused in the alley.
	w.Command(helper, "install ice-breaker")
	if helper.WeaponBonus != 0 {
		t.Fatal("install must require a ripperdoc")
	}
	// To the Night Market (ripperdoc) and install.
	w.Command(helper, "south") // -> the_sprawl
	w.Command(helper, "south") // -> market (ripperdoc)
	w.Command(helper, "install ice-breaker")
	if helper.WeaponBonus != 5 || helper.WeaponName != "ice-breaker" || helper.Inv["ice-breaker"] != 0 {
		t.Fatalf("ripperdoc install failed: bonus=%d inv=%v", helper.WeaponBonus, helper.Inv)
	}

	// Give a stimpak back to the victim (route the victim to the market first).
	w.Command(victim, "out")
	w.Command(victim, "east")
	w.Command(victim, "south")
	if victim.RoomID != "market" {
		t.Fatalf("victim should reach the market, at %s", victim.RoomID)
	}
	w.Command(helper, "give stimpak Case")
	if victim.Inv["stimpak"] != 1 || helper.Inv["stimpak"] != 2 {
		t.Fatalf("give should transfer one stimpak: victim=%d helper=%d", victim.Inv["stimpak"], helper.Inv["stimpak"])
	}
}
