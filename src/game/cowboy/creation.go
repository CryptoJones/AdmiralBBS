package cowboy

import (
	"bufio"
	"strconv"
	"strings"
)

// SkillPoints is the pool a new cowboy distributes across attributes during
// character creation, on top of their class's base stats.
const SkillPoints = 6

// Class is a starting archetype (Cyberpunk 2020 / GURPS-flavored). Each sets
// base Body/Reflexes/Intelligence; the player then spends SkillPoints.
type Class struct {
	ID                           string
	Name                         string
	Desc                         string
	Body, Reflexes, Intelligence int
}

var classes = []Class{
	{ID: "hacker", Name: "Hacker", Desc: "elite breach artist — lethal in the Net (high INT)", Body: 8, Reflexes: 9, Intelligence: 13},
	{ID: "enforcer", Name: "Enforcer", Desc: "street muscle — wrecks meatspace foes (high BODY)", Body: 13, Reflexes: 11, Intelligence: 6},
	{ID: "operator", Name: "Operator", Desc: "fast and slippery — quick on the draw (high REFLEXES)", Body: 9, Reflexes: 13, Intelligence: 8},
	{ID: "mechanic", Name: "Mechanic", Desc: "gearhead generalist — balanced across the board", Body: 10, Reflexes: 10, Intelligence: 10},
}

// Classes returns the selectable archetypes.
func Classes() []Class { return append([]Class(nil), classes...) }

func classByID(id string) (Class, bool) {
	for _, c := range classes {
		if c.ID == id {
			return c, true
		}
	}
	return Class{}, false
}

// CharSpec is the result of character creation: chosen class plus final stats.
type CharSpec struct {
	ClassID                      string
	Body, Reflexes, Intelligence int
}

// RunCharacterCreation drives the interactive new-character flow over a raw
// terminal: pick a class, then spend SkillPoints across the three attributes.
// It is pure I/O over r/out (echo through out), so it's unit-testable by feeding
// bytes. Invalid input falls back to sensible defaults rather than erroring.
func RunCharacterCreation(r *bufio.Reader, out func(string)) (CharSpec, error) {
	out(crlf + style(neon, "== CHARACTER CREATION ==") + crlf)
	out(style(dim, "New runner detected. Build your cowboy.") + crlf + crlf)
	for i, c := range classes {
		out("  " + style(gold, itoa(i+1)+")") + " " + style(green, c.Name) +
			style(dim, "  [B"+itoa(c.Body)+" R"+itoa(c.Reflexes)+" I"+itoa(c.Intelligence)+"] — "+c.Desc) + crlf)
	}
	out(crlf + style(green, "Choose a class [1-"+itoa(len(classes))+"]: "))
	line, err := ReadLine(r, out)
	if err != nil {
		return CharSpec{}, err
	}
	chosen := classes[0]
	if n, e := strconv.Atoi(strings.TrimSpace(line)); e == nil && n >= 1 && n <= len(classes) {
		chosen = classes[n-1]
	}
	out(style(neon, "You are a "+chosen.Name+".") + crlf)

	b, rfx, in := chosen.Body, chosen.Reflexes, chosen.Intelligence
	remaining := SkillPoints
	out(crlf + style(gold, "You have "+itoa(SkillPoints)+" skill points to spend.") + crlf)

	addBody, err := askPoints(r, out, "BODY (melee damage, HP)", remaining)
	if err != nil {
		return CharSpec{}, err
	}
	b += addBody
	remaining -= addBody

	addRfx, err := askPoints(r, out, "REFLEXES (to-hit, dodge)", remaining)
	if err != nil {
		return CharSpec{}, err
	}
	rfx += addRfx
	remaining -= addRfx

	// Whatever's left flows into Intelligence (netrun breaching).
	in += remaining
	if remaining > 0 {
		out(style(dim, itoa(remaining)+" leftover point(s) routed to INTELLIGENCE.") + crlf)
	}

	out(crlf + style(neon, "Final loadout — ") +
		"BODY " + itoa(b) + "  REFLEXES " + itoa(rfx) + "  INTELLIGENCE " + itoa(in) + crlf)
	return CharSpec{ClassID: chosen.ID, Body: b, Reflexes: rfx, Intelligence: in}, nil
}

func askPoints(r *bufio.Reader, out func(string), label string, remaining int) (int, error) {
	if remaining <= 0 {
		return 0, nil
	}
	out(style(green, "Points into "+label+" (0-"+itoa(remaining)+"): "))
	line, err := ReadLine(r, out)
	if err != nil {
		return 0, err
	}
	n, e := strconv.Atoi(strings.TrimSpace(line))
	if e != nil || n < 0 {
		n = 0
	}
	if n > remaining {
		n = remaining
	}
	return n, nil
}
