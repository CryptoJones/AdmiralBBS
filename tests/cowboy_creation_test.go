package tests

import (
	"bufio"
	"strings"
	"testing"

	"admiralbbs/src/game/cowboy"
)

func TestCowboyCharacterCreation(t *testing.T) {
	// Class 2 = Solo (B13 R11 I6); spend 3 into BODY, 1 into REFLEXES, rest (2) to INT.
	in := bufio.NewReader(strings.NewReader("2\r\n3\r\n1\r\n"))
	var out strings.Builder
	spec, err := cowboy.RunCharacterCreation(in, func(s string) { out.WriteString(s) })
	if err != nil {
		t.Fatal(err)
	}
	if spec.ClassID != "solo" {
		t.Fatalf("class = %q, want solo", spec.ClassID)
	}
	if spec.Body != 16 || spec.Reflexes != 12 || spec.Intelligence != 8 {
		t.Fatalf("stats = B%d R%d I%d, want B16 R12 I8", spec.Body, spec.Reflexes, spec.Intelligence)
	}
	s := out.String()
	if !strings.Contains(s, "CHARACTER CREATION") || !strings.Contains(s, "Netrunner") || !strings.Contains(s, "Solo") {
		t.Error("creation screen should list the classes")
	}
}

func TestCowboyCreationDefaultsOnGarbage(t *testing.T) {
	// Bad class + non-numeric point entries -> netrunner, no points spent.
	in := bufio.NewReader(strings.NewReader("99\r\nxyz\r\nnope\r\n"))
	spec, err := cowboy.RunCharacterCreation(in, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if spec.ClassID != "netrunner" {
		t.Fatalf("garbage class should default to netrunner, got %q", spec.ClassID)
	}
	// Netrunner base is B8 R9 I13; garbage allocations add 0 to B/R, leftover to I.
	if spec.Body != 8 || spec.Reflexes != 9 || spec.Intelligence != 13+cowboy.SkillPoints {
		t.Fatalf("unexpected defaulted stats: %+v", spec)
	}
}

func TestCowboyCreateCharacterAppliesClassAndPersists(t *testing.T) {
	store := cowboy.NewMemStore()
	w := cowboy.NewWorld(store)
	out, _ := sink()
	spec := cowboy.CharSpec{ClassID: "solo", Body: 16, Reflexes: 12, Intelligence: 8}

	if w.HasCharacter("Rogue") {
		t.Fatal("brand-new name should not exist yet")
	}
	p := w.CreateCharacter("Rogue", spec, out)
	if p.Class != "Solo" || p.Body != 16 || p.Reflexes != 12 || p.Intelligence != 8 {
		t.Fatalf("class/stats not applied: %+v", p)
	}
	if p.HP <= 40 { // higher Body -> higher MaxHP than the default 40
		t.Fatalf("MaxHP should reflect higher Body, got HP=%d", p.HP)
	}
	// Persisted immediately, and a later login reloads the class.
	if !w.HasCharacter("Rogue") {
		t.Fatal("created character should be saved")
	}
	w.Disconnect(p)
	w2 := cowboy.NewWorld(store)
	out2, _ := sink()
	p2 := w2.Connect("Rogue", out2)
	if p2.Class != "Solo" || p2.Body != 16 {
		t.Fatalf("returning login lost class/stats: %+v", p2)
	}
}
