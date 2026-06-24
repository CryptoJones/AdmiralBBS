package cowboy

import "strings"

// program is a netrun demon: a loadable, RAM-costed ability run with RUN <name>.
// Damage programs are deliberate exploits — they auto-land (no to-hit roll) but
// still respect the target's armor. Every netrunner carries the stock toolkit.
type program struct {
	id      string
	name    string
	ram     int
	netOnly bool
	desc    string
	kind    string // "damage" | "leech" | "shield" | "repair"
	power   int    // base magnitude added to an Intelligence term
}

var programs = []program{
	{id: "scalpel", name: "Scalpel", ram: 2, netOnly: true, kind: "damage", power: 4,
		desc: "precise breach: INT+4 damage to your target"},
	{id: "hammer", name: "Hammer", ram: 4, netOnly: true, kind: "damage", power: 12,
		desc: "heavy breach: INT+12 damage to your target"},
	{id: "leech", name: "Leech", ram: 5, netOnly: true, kind: "leech", power: 6,
		desc: "drain: INT+6 damage and heal half of it"},
	{id: "mirror", name: "Mirror", ram: 3, netOnly: false, kind: "shield", power: 3,
		desc: "deflector: reduce incoming damage for 3 rounds"},
	{id: "medic", name: "Medic", ram: 3, netOnly: false, kind: "repair", power: 15,
		desc: "repair routine: restore HP (works anywhere)"},
}

func programByID(id string) (program, bool) {
	for _, pr := range programs {
		if pr.id == id {
			return pr, true
		}
	}
	return program{}, false
}

// listPrograms shows the deck's loaded demons (available via SCORE/HELP too).
func (w *World) listPrograms(p *Player) {
	p.send(style(neon, "-- Loaded programs (RUN <name>) --") + crlf)
	for _, pr := range programs {
		tag := ""
		if pr.netOnly {
			tag = style(dim, " [Net only]")
		}
		p.send("  " + style(green, strings.ToLower(pr.id)) + style(gold, " ("+itoa(pr.ram)+" RAM)") +
			style(dim, " — "+pr.desc) + tag + crlf)
	}
}

// run executes a netrun program for its RAM cost.
func (w *World) run(p *Player, arg string) {
	name := strings.ToLower(strings.TrimSpace(arg))
	if name == "" || name == "list" {
		w.listPrograms(p)
		return
	}
	pr, ok := programByID(name)
	if !ok {
		p.send(style(dim, "No such program. RUN with no name lists them.") + crlf)
		return
	}
	if pr.netOnly && !w.inNet(p) {
		p.send(style(dim, pr.name+" only runs inside the Net.") + crlf)
		return
	}
	if p.RAM < pr.ram {
		p.send(style(red, "Not enough RAM for "+pr.name+" (need "+itoa(pr.ram)+").") + crlf)
		return
	}

	switch pr.kind {
	case "shield":
		p.RAM -= pr.ram
		p.shieldTicks = 3
		p.shieldAmt = pr.power + p.Intelligence/4
		p.send(style(green, "You compile Mirror — incoming damage cut by "+itoa(p.shieldAmt)+" for 3 rounds.") + crlf)
	case "repair":
		p.RAM -= pr.ram
		heal := pr.power + p.Intelligence/2
		p.HP += heal
		if p.HP > p.MaxHP {
			p.HP = p.MaxHP
		}
		p.send(style(green, "You run Medic — HP now "+itoa(p.HP)+"/"+itoa(p.MaxHP)+".") + crlf)
	case "damage", "leech":
		w.runDamage(p, pr)
	}
}

func (w *World) runDamage(p *Player, pr program) {
	power := pr.power + p.Intelligence
	// Target the current mob or duel opponent.
	if m := p.fighting; m != nil && !m.dead && m.RoomID == p.RoomID {
		p.RAM -= pr.ram
		hit := dmg(power, m.tmpl.AC)
		m.HP -= hit
		p.send(style(hot, "You execute "+pr.name+" on "+m.tmpl.Name+" for "+itoa(hit)+"!") + crlf)
		if pr.kind == "leech" {
			w.heal(p, hit/2)
			p.send(style(green, "Leech siphons "+itoa(hit/2)+" HP to you.") + crlf)
		}
		if m.HP <= 0 {
			w.killMob(p, m)
		}
		return
	}
	if d := p.pvpTarget; d != nil && w.inNet(p) && d.RoomID == p.RoomID {
		p.RAM -= pr.ram
		hit := dmg(power, playerAC(d))
		d.HP -= hit
		p.send(style(hot, "You execute "+pr.name+" on "+d.Name+" for "+itoa(hit)+"!") + crlf)
		d.send(style(red, p.Name+" hits you with "+pr.name+" for "+itoa(hit)+"!") + crlf)
		if pr.kind == "leech" {
			w.heal(p, hit/2)
		}
		if d.HP <= 0 {
			w.pvpFlatline(p, d)
		}
		return
	}
	p.send(style(dim, pr.name+" needs a target — engage something first.") + crlf)
}

func (w *World) heal(p *Player, amt int) {
	p.HP += amt
	if p.HP > p.MaxHP {
		p.HP = p.MaxHP
	}
}

// applyShield reduces one incoming hit by the player's active shield (min 1).
func applyShield(p *Player, d int) int {
	if p.shieldTicks > 0 && p.shieldAmt > 0 {
		d -= p.shieldAmt
		if d < 1 {
			d = 1
		}
	}
	return d
}
