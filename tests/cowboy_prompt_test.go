package tests

import (
	"testing"

	"admiralbbs/src/game/cowboy"
)

// An idle player must NOT have the prompt re-printed every tick (the bug: the
// prompt repeated while you sat reading). PromptIfDirty is a no-op when nothing
// was sent since the last prompt.
func TestCowboyIdleDoesNotRepeatPrompt(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	out, buf := sink()
	p := w.Connect("Case", out) // neon_alley — no mobs, not in combat
	w.Command(p, "look")        // produces output + one prompt, clears "dirty"

	before := buf.Len()
	for i := 0; i < 5; i++ {
		w.Tick()
		w.PromptIfDirty(p)
	}
	if buf.Len() != before {
		t.Fatalf("idle ticks re-emitted output/prompt: grew by %d bytes", buf.Len()-before)
	}
}

// During combat, a tick DOES produce output, so the prompt should refresh.
func TestCowboyCombatRepromptsOnTick(t *testing.T) {
	w := cowboy.NewWorld(cowboy.NewMemStore())
	w.SetRoll(alwaysHit)
	out, buf := sink()
	p := w.Connect("Case", out)
	w.Command(p, "out")             // re-sleeve bay -> neon_alley
	w.Command(p, "east")            // the_sprawl
	w.Command(p, "north")           // back_alley (a ganger)
	w.Command(p, "attack ganger")   // engage; clears dirty at the prompt

	before := buf.Len()
	w.Tick()             // combat round -> output to the player
	w.PromptIfDirty(p)   // ...so a fresh prompt should follow
	if buf.Len() <= before {
		t.Fatal("a combat tick should produce output and refresh the prompt")
	}
}
