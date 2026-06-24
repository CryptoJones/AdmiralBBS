package cowboy

import "strings"

// engage targets a hostile (a mob, or — in the Net — another runner for PvP) and
// starts a fight. Rounds resolve on Tick, MajorMUD-style.
func (w *World) engage(p *Player, arg string) {
	arg = strings.ToLower(strings.TrimSpace(arg))

	// In the Net, you can jack another runner: PvP.
	if w.inNet(p) && arg != "" {
		if target := w.playerInRoomByName(p.RoomID, arg, p); target != nil {
			w.engagePvP(p, target)
			return
		}
	}

	mobs := w.liveMobsIn(p.RoomID)
	if len(mobs) == 0 {
		p.send(style(dim, "Nothing here to fight.") + crlf)
		return
	}
	var target *Mob
	if arg == "" {
		target = mobs[0]
	} else {
		for _, m := range mobs {
			if strings.Contains(strings.ToLower(m.tmpl.Name), arg) || strings.Contains(m.tmpl.ID, arg) {
				target = m
				break
			}
		}
	}
	if target == nil {
		p.send(style(dim, "You don't see '"+arg+"' here.") + crlf)
		return
	}
	p.pvpTarget = nil // mob combat supersedes any duel
	p.fighting = target
	if target.target == nil {
		target.target = p
	}
	verb := "You lunge at "
	if w.inNet(p) {
		verb = "You jack a breach protocol into "
	}
	p.send(style(hot, verb+target.tmpl.Name+"!") + crlf)
	w.broadcast(p.RoomID, p, style(dim, p.Name+" attacks "+target.tmpl.Name+".")+crlf)
}

func (w *World) engagePvP(p, target *Player) {
	if target == p {
		p.send(style(dim, "You can't jack yourself.") + crlf)
		return
	}
	p.fighting = nil
	p.pvpTarget = target
	p.send(style(hot, "You jack into "+target.Name+"'s deck — netrun duel!") + crlf)
	target.send(style(hot, p.Name+" jacks into your deck — defend yourself!") + crlf)
	w.broadcast(p.RoomID, p, style(dim, p.Name+" and "+target.Name+" are duelling in the Net.")+crlf)
}

func (w *World) playerInRoomByName(roomID, name string, except *Player) *Player {
	name = strings.ToLower(name)
	for _, o := range w.playersIn(roomID, except) {
		if strings.ToLower(o.Name) == name || strings.HasPrefix(strings.ToLower(o.Name), name) {
			return o
		}
	}
	return nil
}

// flee attempts to break combat (mob or duel) and bolt to a random exit.
func (w *World) flee(p *Player) {
	if p.fighting == nil && p.pvpTarget == nil {
		p.send(style(dim, "You're not in combat.") + crlf)
		return
	}
	if w.roll(2) != 0 {
		p.send(style(red, "You can't break the connection — the fight holds you!") + crlf)
		return
	}
	if p.fighting != nil && p.fighting.target == p {
		p.fighting.target = nil
	}
	p.fighting = nil
	p.pvpTarget = nil
	r := w.room(p.RoomID)
	var exits []string
	for _, d := range []string{"north", "south", "east", "west", "up", "down"} {
		if _, ok := r.Exits[d]; ok {
			exits = append(exits, d)
		}
	}
	p.send(style(green, "You rip free and bolt!") + crlf)
	if len(exits) > 0 {
		dir := exits[w.roll(len(exits))]
		w.broadcast(p.RoomID, p, style(dim, p.Name+" flees "+dir+".")+crlf)
		p.RoomID = r.Exits[dir]
		w.broadcast(p.RoomID, p, style(dim, p.Name+" skids in, breathless.")+crlf)
		w.lookText(p)
	}
}

// Tick advances the world one round: aggro, mob fights, PvP duels, deaths,
// respawns, and out-of-combat regen (HP and RAM).
func (w *World) Tick() {
	w.aggro()
	w.resolveCombat()
	w.resolvePvP()
	w.respawnDead()
	w.regen()
}

func (w *World) aggro() {
	for _, m := range w.mobs {
		if m.dead || !m.tmpl.Aggressive || m.target != nil {
			continue
		}
		victims := w.playersIn(m.RoomID, nil)
		if len(victims) == 0 {
			continue
		}
		v := victims[w.roll(len(victims))]
		m.target = v
		if v.fighting == nil && v.pvpTarget == nil {
			v.fighting = m
		}
		v.send(style(hot, m.tmpl.Name+" locks onto you and attacks!") + crlf)
	}
}

// playerSwing returns the player's damage for one round, route-aware. In the Net
// a breach spends 1 RAM for full Intelligence-powered damage; with no RAM the
// breach sputters at half strength.
func (w *World) playerSwing(p *Player) int {
	atk := w.effAttack(p)
	if w.inNet(p) {
		if p.RAM > 0 {
			p.RAM--
		} else {
			atk /= 2
			if atk < 1 {
				atk = 1
			}
			p.send(style(dim, "Low RAM — your breach sputters at half power.") + crlf)
		}
	}
	return atk
}

func (w *World) resolveCombat() {
	for _, p := range w.players {
		m := p.fighting
		if m == nil {
			continue
		}
		if m.dead || m.RoomID != p.RoomID {
			p.fighting = nil
			continue
		}
		if w.toHit(p.Reflexes, m.tmpl.AC) {
			d := dmg(w.playerSwing(p), m.tmpl.AC)
			m.HP -= d
			p.send(style(green, "You hit "+m.tmpl.Name+" for "+itoa(d)+".") + crlf)
		} else {
			p.send(style(dim, "You miss "+m.tmpl.Name+".") + crlf)
		}
		if m.HP <= 0 {
			w.killMob(p, m)
			continue
		}
		if m.target == p {
			if w.toHit(m.tmpl.Damage/2, playerAC(p)) {
				d := dmg(m.tmpl.Damage, playerAC(p))
				p.HP -= d
				p.send(style(red, m.tmpl.Name+" hits you for "+itoa(d)+".") + crlf)
				if p.HP <= 0 {
					w.flatline(p, m)
				}
			} else {
				p.send(style(dim, m.tmpl.Name+" misses you.") + crlf)
			}
		}
	}
}

// resolvePvP runs one round of every active netrun duel. Both runners swing in
// the same tick (each processed on their turn).
func (w *World) resolvePvP() {
	for _, p := range w.players {
		d := p.pvpTarget
		if d == nil {
			continue
		}
		if !w.inNet(p) || d.RoomID != p.RoomID || w.players[d.ID] == nil {
			p.pvpTarget = nil
			p.send(style(dim, "Your duel target is gone.") + crlf)
			continue
		}
		if w.toHit(p.Reflexes, playerAC(d)) {
			hit := dmg(w.playerSwing(p), playerAC(d))
			d.HP -= hit
			p.send(style(green, "You breach "+d.Name+"'s deck for "+itoa(hit)+".") + crlf)
			d.send(style(red, p.Name+" breaches your deck for "+itoa(hit)+".") + crlf)
			if d.HP <= 0 {
				w.pvpFlatline(p, d)
			}
		} else {
			p.send(style(dim, d.Name+" slips your breach.") + crlf)
		}
	}
}

func (w *World) killMob(p *Player, m *Mob) {
	// Multi-stage ICE: morph into the next, harder form instead of dying.
	if m.tmpl.Next != "" {
		if next, ok := w.tmpls[m.tmpl.Next]; ok {
			old := m.tmpl.Name
			m.tmpl = next
			m.HP = next.HP
			m.target = p
			p.fighting = m
			p.send(style(hot, "The ICE reconfigures! "+old+" collapses into "+next.Name+"!") + crlf)
			w.broadcast(p.RoomID, p, style(dim, old+" reconfigures into "+next.Name+".")+crlf)
			return
		}
	}
	m.dead = true
	m.HP = 0
	m.respawnIn = w.respawnTicks
	if m.target != nil {
		m.target.fighting = nil
		m.target = nil
	}
	p.fighting = nil
	p.XP += m.tmpl.XP
	p.Eddies += m.tmpl.Eddies
	p.send(style(hot, "*** "+m.tmpl.Name+" is destroyed! ***") + crlf)
	p.send(style(gold, "You gain "+itoa(m.tmpl.XP)+" XP and €$"+itoa(m.tmpl.Eddies)+" eddies.") + crlf)
	w.broadcast(p.RoomID, p, style(dim, p.Name+" destroys "+m.tmpl.Name+".")+crlf)
	w.creditQuestKill(p, m.tmpl.ID)
	w.checkLevelUp(p)
}

// flatline handles player death by a mob: half HP, respawn at the start, and a
// credit/XP penalty — never permadeath.
func (w *World) flatline(p *Player, killer *Mob) {
	p.send(style(red, "*** FLATLINE — your deck browns out and dumps you back to the alley. ***") + crlf)
	if killer.target == p {
		killer.target = nil
	}
	p.fighting = nil
	p.pvpTarget = nil
	w.respawnPlayer(p, p.Eddies/10)
}

// pvpFlatline handles losing a netrun duel: the winner siphons a cut of the
// loser's eddies (data theft) and the loser respawns in meatspace.
func (w *World) pvpFlatline(winner, loser *Player) {
	loot := loser.Eddies / 10
	loser.send(style(red, "*** YOUR DECK IS FRIED — "+winner.Name+" flatlines you and siphons €$"+itoa(loot)+". ***") + crlf)
	winner.send(style(gold, "*** You fry "+loser.Name+"'s deck and siphon €$"+itoa(loot)+"! ***") + crlf)
	winner.Eddies += loot
	winner.pvpTarget = nil
	loser.pvpTarget = nil
	loser.fighting = nil
	w.respawnPlayer(loser, loot)
}

// respawnPlayer sends a defeated player back to the start at half HP, docking
// `lostEddies` (already credited elsewhere if stolen) and a little XP.
func (w *World) respawnPlayer(p *Player, lostEddies int) {
	p.Eddies -= lostEddies
	if p.Eddies < 0 {
		p.Eddies = 0
	}
	p.XP -= xpToNext(p.Level) / 10
	if p.XP < 0 {
		p.XP = 0
	}
	p.HP = p.MaxHP/2 + 1
	p.RoomID = startRoom
	w.lookText(p)
}

func (w *World) respawnDead() {
	for _, m := range w.mobs {
		if !m.dead {
			continue
		}
		m.respawnIn--
		if m.respawnIn <= 0 {
			m.dead = false
			m.HP = m.tmpl.HP
			m.RoomID = m.tmpl.Home
			m.target = nil
			w.broadcast(m.RoomID, nil, style(dim, m.tmpl.Name+" reinitializes.")+crlf)
		}
	}
}

func (w *World) regen() {
	for _, p := range w.players {
		inCombat := p.fighting != nil || p.pvpTarget != nil
		if inCombat {
			continue
		}
		if p.HP < p.MaxHP {
			heal := p.MaxHP / 20
			if heal < 1 {
				heal = 1
			}
			p.HP += heal
			if p.HP > p.MaxHP {
				p.HP = p.MaxHP
			}
		}
		if mr := maxRAM(p); p.RAM < mr {
			p.RAM++
		}
	}
}
