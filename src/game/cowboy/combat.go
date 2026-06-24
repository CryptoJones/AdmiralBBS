package cowboy

import "strings"

// engage targets a hostile in the room and starts a fight (rounds resolve on
// Tick, MajorMUD-style — up to death or flee).
func (w *World) engage(p *Player, arg string) {
	arg = strings.ToLower(strings.TrimSpace(arg))
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

// flee attempts to break combat and bolt to a random exit.
func (w *World) flee(p *Player) {
	if p.fighting == nil {
		p.send(style(dim, "You're not in combat.") + crlf)
		return
	}
	if w.roll(2) != 0 {
		p.send(style(red, "You can't break the connection — the fight holds you!") + crlf)
		return
	}
	mob := p.fighting
	p.fighting = nil
	if mob.target == p {
		mob.target = nil
	}
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

// Tick advances the world one combat round: aggro, fights, deaths, respawns,
// and out-of-combat regen. The server calls this on a fixed interval.
func (w *World) Tick() {
	w.aggro()
	w.resolveCombat()
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
		if v.fighting == nil {
			v.fighting = m
		}
		v.send(style(hot, m.tmpl.Name+" locks onto you and attacks!") + crlf)
	}
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
		// Player's swing.
		if w.toHit(p.Reflexes, m.tmpl.AC) {
			d := dmg(w.effAttack(p), m.tmpl.AC)
			m.HP -= d
			p.send(style(green, "You hit "+m.tmpl.Name+" for "+itoa(d)+".") + crlf)
		} else {
			p.send(style(dim, "You miss "+m.tmpl.Name+".") + crlf)
		}
		if m.HP <= 0 {
			w.killMob(p, m)
			continue
		}
		// Mob's counter (only against the player it's locked on).
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

func (w *World) killMob(p *Player, m *Mob) {
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

// flatline handles player death: half HP, respawn at the start, and a credit/XP
// penalty — never permadeath (it's a BBS door, callers come back).
func (w *World) flatline(p *Player, killer *Mob) {
	p.send(style(red, "*** FLATLINE — your deck browns out and dumps you back to the alley. ***") + crlf)
	if killer.target == p {
		killer.target = nil
	}
	p.fighting = nil
	lostEddies := p.Eddies / 10
	p.Eddies -= lostEddies
	p.XP -= xpToNext(p.Level) / 10
	if p.XP < 0 {
		p.XP = 0
	}
	p.HP = p.MaxHP/2 + 1
	p.RoomID = startRoom
	if lostEddies > 0 {
		p.send(style(dim, "You lost €$"+itoa(lostEddies)+" and some XP in the crash.") + crlf)
	}
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
		if p.fighting != nil || p.HP >= p.MaxHP {
			continue
		}
		heal := p.MaxHP / 20
		if heal < 1 {
			heal = 1
		}
		p.HP += heal
		if p.HP > p.MaxHP {
			p.HP = p.MaxHP
		}
	}
}
