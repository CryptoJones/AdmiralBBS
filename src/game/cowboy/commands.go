package cowboy

import "strings"

// netRooms are inside the Net: here your attack is a netrun BREACH driven by
// Intelligence, not a meatspace strike driven by Body.
var netRooms = map[string]bool{"the_net": true, "deep_net": true}

func (w *World) inNet(p *Player) bool { return netRooms[p.RoomID] }

// effAttack is the player's damage this round, route-dependent (breach vs melee).
func (w *World) effAttack(p *Player) int {
	if w.inNet(p) {
		return 3 + p.Intelligence/2 + p.Level + p.WeaponBonus
	}
	return p.attack()
}

var dirAliases = map[string]string{
	"n": "north", "s": "south", "e": "east", "w": "west", "u": "up", "d": "down",
	"north": "north", "south": "south", "east": "east", "west": "west", "up": "up", "down": "down",
}

// Command parses and executes a single input line for player p. It returns true
// if the player asked to quit (the server then disconnects them).
func (w *World) Command(p *Player, line string) (quit bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		w.sendPrompt(p)
		return false
	}
	fields := strings.Fields(line)
	cmd := strings.ToLower(fields[0])
	arg := strings.TrimSpace(strings.TrimPrefix(line, fields[0]))

	if dir, ok := dirAliases[cmd]; ok {
		w.move(p, dir)
		w.sendPrompt(p)
		return false
	}

	switch cmd {
	case "look", "l":
		w.lookText(p)
	case "say", "'":
		w.say(p, arg)
	case "who":
		w.who(p)
	case "score", "stats", "st", "sc":
		w.score(p)
	case "attack", "kill", "k", "breach":
		w.engage(p, arg)
	case "flee", "jackout", "disconnect":
		w.flee(p)
	case "list", "shop":
		w.list(p)
	case "buy":
		w.buy(p, arg)
	case "use":
		w.use(p, arg)
	case "inventory", "inv", "i":
		w.inventory(p)
	case "quests", "missions", "bounties":
		w.showQuests(p)
	case "accept", "take":
		w.accept(p, arg)
	case "claim", "turnin":
		w.claim(p)
	case "help", "?", "commands":
		p.send(helpText())
	case "quit", "logout", "exit":
		p.send(style(neon, "Jacking out. The grid forgets you... for now.") + crlf)
		return true
	default:
		p.send(style(dim, "Unknown command. Type HELP.") + crlf)
	}
	w.sendPrompt(p)
	return false
}

// Prompt re-displays the player's status prompt (used by the server after a
// tick so combat output doesn't leave a dangling line).
func (w *World) Prompt(p *Player) { w.sendPrompt(p) }

func (w *World) sendPrompt(p *Player) {
	hpColor := green
	if p.HP*3 < p.MaxHP {
		hpColor = red
	}
	mode := "MEAT"
	if w.inNet(p) {
		mode = "NET"
	}
	p.send(style(hpColor, "["+itoa(p.HP)+"/"+itoa(p.MaxHP)+"hp]") +
		style(dim, " ["+mode+"] ") + style(green, "> "))
}

func (w *World) lookText(p *Player) {
	r := w.room(p.RoomID)
	if r == nil {
		p.send(style(red, "You are nowhere. (corrupted location)") + crlf)
		return
	}
	p.send(crlf + style(neon, r.Name) + crlf + r.Desc + crlf)
	if r.Vendor {
		p.send(style(gold, "A vendor terminal hums here. Type LIST.") + crlf)
	}
	// Exits.
	var dirs []string
	for _, d := range []string{"north", "south", "east", "west", "up", "down"} {
		if _, ok := r.Exits[d]; ok {
			dirs = append(dirs, d)
		}
	}
	p.send(style(dim, "Exits: "+strings.Join(dirs, ", ")) + crlf)
	// Other players.
	for _, other := range w.playersIn(p.RoomID, p) {
		p.send(style(green, other.Name+" is here.") + crlf)
	}
	// Mobs.
	for _, m := range w.liveMobsIn(p.RoomID) {
		p.send(style(hot, m.tmpl.Name+" is here.") + crlf)
	}
}

func (w *World) say(p *Player, msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		p.send(style(dim, "Say what?") + crlf)
		return
	}
	p.send(style(green, "You say: ") + msg + crlf)
	w.broadcast(p.RoomID, p, style(green, p.Name+" says: ")+msg+crlf)
}

func (w *World) who(p *Player) {
	p.send(style(neon, "-- Jacked in right now --") + crlf)
	for _, o := range w.players {
		cls := o.Class
		if cls != "" {
			cls = " " + cls
		}
		p.send("  " + style(green, o.Name) + style(dim, "  (level "+itoa(o.Level)+cls+")") + crlf)
	}
}

func (w *World) score(p *Player) {
	class := p.Class
	if class == "" {
		class = "console cowboy"
	}
	p.send(crlf + style(neon, "== "+p.Name+" :: "+class+" ==") + crlf)
	xpLine := "  Level " + itoa(p.Level) + "   XP " + itoa(p.XP) + "/" + itoa(xpToNext(p.Level))
	if p.Level >= MaxLevel {
		xpLine = "  Level " + itoa(p.Level) + " " + style(gold, "(MAX)")
	}
	p.send(xpLine + crlf)
	p.send("  HP " + itoa(p.HP) + "/" + itoa(p.MaxHP) + "   AC " + itoa(playerAC(p)) + crlf)
	p.send("  Body " + itoa(p.Body) + "   Reflexes " + itoa(p.Reflexes) + "   Intelligence " + itoa(p.Intelligence) + crlf)
	weapon := "bare fists"
	if p.WeaponName != "" {
		weapon = p.WeaponName + " (+" + itoa(p.WeaponBonus) + " atk)"
	}
	p.send("  Weapon: " + weapon + crlf)
	p.send(style(gold, "  €$ "+itoa(p.Eddies)+" eddies") + crlf)
}

func (w *World) inventory(p *Player) {
	p.send(style(neon, "-- Inventory --") + crlf)
	if len(p.Inv) == 0 {
		p.send(style(dim, "  (empty)") + crlf)
		return
	}
	for name, qty := range p.Inv {
		p.send("  " + name + " x" + itoa(qty) + crlf)
	}
}

func (w *World) move(p *Player, dir string) {
	if p.fighting != nil {
		p.send(style(hot, "You're in combat! Break the connection with FLEE first.") + crlf)
		return
	}
	r := w.room(p.RoomID)
	dest, ok := r.Exits[dir]
	if !ok {
		p.send(style(dim, "You can't go "+dir+".") + crlf)
		return
	}
	w.broadcast(p.RoomID, p, style(dim, p.Name+" heads "+dir+".")+crlf)
	p.RoomID = dest
	w.broadcast(p.RoomID, p, style(dim, p.Name+" arrives.")+crlf)
	w.lookText(p)
}

func helpText() string {
	return crlf + style(neon, "== Console Cowboy 2026 — commands ==") + crlf +
		"  Movement : N S E W U D  (or north/south/...)\r\n" +
		"  look (l)        — examine your location\r\n" +
		"  attack <foe>    — engage a hostile (alias kill/breach)\r\n" +
		"  flee            — try to break a fight and bolt\r\n" +
		"  say <msg>       — talk to others in the room\r\n" +
		"  who             — who's jacked in\r\n" +
		"  score (st)      — your character sheet\r\n" +
		"  list / buy <x>  — vendor (at shops); use <item> to consume\r\n" +
		"  inventory (i)   — what you're carrying\r\n" +
		"  quests          — fixer bounty board (at a shop); accept <#> / claim\r\n" +
		"  quit            — jack out\r\n" +
		style(dim, "  In the Net, ATTACK breaches ICE using Intelligence, not Body.") + crlf
}
