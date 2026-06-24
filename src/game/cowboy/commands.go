package cowboy

import "strings"

// netRooms are inside the Net: here your attack is a netrun BREACH driven by
// Intelligence, not a meatspace strike driven by Body.
var netRooms = map[string]bool{"the_net": true, "ice_wall": true, "deep_net": true}

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
	"out": "out", "o": "out", // capsule pod -> street
	"in": "in", // street -> your capsule pod
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
	case "loot", "salvage":
		w.loot(p)
	case "install", "ripper":
		w.install(p, arg)
	case "give", "hand":
		w.give(p, arg)
	case "inventory", "inv", "i":
		w.inventory(p)
	case "quests", "missions", "bounties":
		w.showQuests(p)
	case "accept", "take":
		// A bare ACCEPT takes a pending crew invite; ACCEPT <#> claims a bounty.
		if arg == "" && p.partyInvite != nil {
			w.acceptInvite(p)
		} else {
			w.accept(p, arg)
		}
	case "decline":
		w.declineInvite(p)
	case "claim", "turnin":
		w.claim(p)
	case "run", "exec":
		w.run(p, arg)
	case "programs", "demons":
		w.listPrograms(p)
	case "group", "crew":
		w.group(p, arg)
	case "invite":
		w.invite(p, arg)
	case "home", "rest":
		w.goHome(p)
	case "leave", "ungroup":
		w.leaveParty(p)
	case "gsay", "crewchat", "party":
		w.groupChat(p, arg)
	case "leaderboard", "top", "rankings":
		w.leaderboard(p)
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

// Prompt re-displays the player's status prompt (used by the server right after
// a player joins).
func (w *World) Prompt(p *Player) { w.sendPrompt(p) }

// PromptIfDirty re-displays the prompt ONLY if the player received output since
// their last prompt. The server calls this after each world tick so a player who
// saw combat/chat/room output gets a fresh prompt — but an IDLE player does not
// get the prompt re-printed every tick (which would spam it while they read).
func (w *World) PromptIfDirty(p *Player) {
	if p.dirty {
		w.sendPrompt(p)
	}
}

func (w *World) sendPrompt(p *Player) {
	hpColor := green
	if p.HP*3 < p.MaxHP {
		hpColor = red
	}
	mode := "MEAT"
	ram := ""
	if w.inNet(p) {
		mode = "NET"
		ram = style(neon, " ["+itoa(p.RAM)+"/"+itoa(maxRAM(p))+"ram]")
	}
	promptStr := style(hpColor, "["+itoa(p.HP)+"/"+itoa(p.MaxHP)+"hp]") + ram +
		style(dim, " ["+mode+"] ") + style(green, "> ")
	if p.prompter != nil {
		p.prompter(promptStr) // managed-prompt sink (redraws around async output)
	} else {
		p.send(promptStr)
	}
	p.dirty = false // prompt now shown; nothing owed until the next output
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
	if r.Ripper {
		p.send(style(gold, "A ripperdoc's chair waits here. INSTALL salvaged cyberware.") + crlf)
	}
	// Exits.
	var dirs []string
	for _, d := range []string{"north", "south", "east", "west", "up", "down", "in", "out"} {
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
	// Flatlined sleeves (corpses) waiting to be looted.
	for _, c := range w.corpsesIn(p.RoomID) {
		p.send(style(dim, c.Owner+"'s flatlined sleeve lies here. (LOOT)") + crlf)
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
	p.send("  HP " + itoa(p.HP) + "/" + itoa(p.MaxHP) + "   RAM " + itoa(p.RAM) + "/" + itoa(maxRAM(p)) + "   AC " + itoa(playerAC(p)) + crlf)
	p.send("  Body " + itoa(p.Body) + "   Reflexes " + itoa(p.Reflexes) + "   Intelligence " + itoa(p.Intelligence) + crlf)
	weapon := "bare fists"
	if p.WeaponName != "" {
		weapon = p.WeaponName + " (+" + itoa(p.WeaponBonus) + " atk)"
	}
	p.send("  Weapon: " + weapon + crlf)
	deck := "stock deck"
	if p.DeckBonus > 0 {
		deck = "cyberdeck (+" + itoa(p.DeckBonus) + " max RAM)"
	}
	p.send("  Deck: " + deck + crlf)
	p.send(style(gold, "  €$ "+itoa(p.Eddies)+" eddies") + crlf)
	if p.shieldTicks > 0 {
		p.send(style(dim, "  Mirror shield: -"+itoa(p.shieldAmt)+" dmg for "+itoa(p.shieldTicks)+" more round(s)") + crlf)
	}
	if p.party != nil && len(p.party.Members) > 1 {
		p.send(style(dim, "  Crew: "+itoa(len(p.party.Members))+" members (GROUP to view)") + crlf)
	}
	p.send(style(dim, "  Programs: RUN <name> — see PROGRAMS") + crlf)
}

func (w *World) inventory(p *Player) {
	p.send(style(neon, "-- Inventory --") + crlf)
	p.send(style(gold, "  €$ "+itoa(p.Eddies)+" eddies") + crlf)
	if len(p.Inv) == 0 {
		p.send(style(dim, "  (no items)") + crlf)
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

// goHome steps into your capsule pod — but only from the street (Neon Alley), so
// it's a convenience, not a teleport/escape. Like any move, it's combat-blocked.
func (w *World) goHome(p *Player) {
	if p.RoomID == "capsule" {
		p.send(style(dim, "You're already in your pod.") + crlf)
		return
	}
	if p.fighting != nil || p.pvpTarget != nil {
		p.send(style(hot, "You're in combat! Break the connection with FLEE first.") + crlf)
		return
	}
	if p.RoomID != "neon_alley" {
		p.send(style(dim, "Your capsule hotel is off Neon Alley — get to the street, then HOME (or go IN).") + crlf)
		return
	}
	w.move(p, "in")
}

// loot strips every flatlined sleeve in the room into your pack. Items are
// usable immediately; salvaged cyberware must be re-installed at a ripperdoc.
// Open recovery: anyone can loot any sleeve (recover for a crewmate — or swipe it).
func (w *World) loot(p *Player) {
	cs := w.corpsesIn(p.RoomID)
	if len(cs) == 0 {
		p.send(style(dim, "There's no flatlined sleeve to loot here.") + crlf)
		return
	}
	total := 0
	var cyber []string
	for _, c := range cs {
		for name, qty := range c.Loot {
			if qty <= 0 {
				continue
			}
			p.Inv[name] += qty
			total += qty
			if x, ok := findWare(name); ok && (x.bonus > 0 || x.deck > 0) {
				cyber = append(cyber, name)
			}
		}
	}
	w.removeCorpsesIn(p.RoomID)
	if total == 0 {
		p.send(style(dim, "The sleeve is already stripped bare.") + crlf)
		return
	}
	p.send(style(green, "You strip the sleeve — its gear is now in your pack.") + crlf)
	if len(cyber) > 0 {
		p.send(style(neon, "Salvaged cyberware: ") + strings.Join(cyber, ", ") +
			style(dim, " — INSTALL it at a ripperdoc to use it again.") + crlf)
	}
	w.broadcast(p.RoomID, p, style(dim, p.Name+" loots a flatlined sleeve.")+crlf)
}

// give hands an inventory item to another runner in the room (e.g. returning a
// crewmate's recovered gear). Syntax: GIVE <item> <runner>.
func (w *World) give(p *Player, arg string) {
	fields := strings.Fields(arg)
	if len(fields) < 2 {
		p.send(style(dim, "Give what to whom? GIVE <item> <runner>.") + crlf)
		return
	}
	targetName := fields[len(fields)-1]
	item := strings.ToLower(strings.Join(fields[:len(fields)-1], " "))
	target := w.playerInRoomByName(p.RoomID, targetName, p)
	if target == nil {
		p.send(style(dim, "No runner named '"+targetName+"' is here.") + crlf)
		return
	}
	if p.Inv[item] <= 0 {
		p.send(style(dim, "You don't have a "+item+".") + crlf)
		return
	}
	w.consumeInv(p, item)
	target.Inv[item]++
	p.send(style(green, "You hand "+target.Name+" the "+item+".") + crlf)
	target.send(style(green, p.Name+" hands you a "+item+".") + crlf)
}

func helpText() string {
	return crlf + style(neon, "== Console Cowboy 2026 — commands ==") + crlf +
		"  Movement : N S E W U D  (or north/south/...)\r\n" +
		"  home / in / out — your private capsule pod (off Neon Alley); spawn-safe\r\n" +
		"  look (l)        — examine your location\r\n" +
		"  attack <foe>    — engage a hostile (alias kill/breach)\r\n" +
		"  flee            — try to break a fight and bolt\r\n" +
		"  say <msg>       — talk to others in the room\r\n" +
		"  who             — who's jacked in\r\n" +
		"  score (st)      — your character sheet\r\n" +
		"  list / buy <x>  — vendor (at shops); use <item> to consume\r\n" +
		"  loot            — strip a flatlined sleeve (corpse) of its gear\r\n" +
		"  install <cyber> — ripperdoc re-installs salvaged cyberware (at the Night Market)\r\n" +
		"  give <item> <runner> — hand recovered gear back to a crewmate\r\n" +
		"  inventory (i)   — what you're carrying\r\n" +
		"  quests          — fixer bounty board (at a shop); accept <#> / claim\r\n" +
		"  programs / run <name> — netrun demons (scalpel/hammer/leech/mirror/medic)\r\n" +
		"  invite <runner> — invite to your crew (leader only); they ACCEPT/DECLINE\r\n" +
		"  group / crew     — show your crew (shared XP in-room); gsay <msg>; leave\r\n" +
		"  leaderboard     — top runners by level\r\n" +
		"  quit            — jack out\r\n" +
		style(dim, "  In the Net, ATTACK breaches ICE using Intelligence and spends RAM\r\n"+
			"  (buy a cyberdeck for more, ram-chips to refill). PvP is LIVE EVERYWHERE\r\n"+
			"  except the street outside the clone pods — draw on a runner there and a\r\n"+
			"  security drone flatlines you. Die and your sleeve drops with your gear;\r\n"+
			"  LOOT it, re-INSTALL cyberware at a ripperdoc. Some ICE morphs when broken.") + crlf
}
