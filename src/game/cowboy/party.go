package cowboy

import "strings"

// Party is a co-op crew. Members who are in the same room when one of them lands
// a kill split the XP (with a small no-penalty rounding in players' favor).
type Party struct {
	Members []*Player
}

func (pt *Party) has(p *Player) bool {
	for _, m := range pt.Members {
		if m == p {
			return true
		}
	}
	return false
}

func (pt *Party) add(p *Player)    { pt.Members = append(pt.Members, p); p.party = pt }
func (pt *Party) remove(p *Player) {
	out := pt.Members[:0]
	for _, m := range pt.Members {
		if m != p {
			out = append(out, m)
		}
	}
	pt.Members = out
	p.party = nil
}

func (pt *Party) broadcast(msg string) {
	for _, m := range pt.Members {
		m.send(msg)
	}
}

// group with no arg shows the crew; `group <name>` recruits an online runner.
func (w *World) group(p *Player, arg string) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		w.showParty(p)
		return
	}
	target := w.byName[strings.ToLower(arg)]
	if target == nil {
		p.send(style(dim, "No runner named '"+arg+"' is jacked in.") + crlf)
		return
	}
	if target == p {
		p.send(style(dim, "You can't crew up with yourself.") + crlf)
		return
	}
	if target.party != nil {
		p.send(style(dim, target.Name+" is already in a crew.") + crlf)
		return
	}
	if p.party == nil {
		p.party = &Party{}
		p.party.add(p)
	}
	p.party.add(target)
	p.party.broadcast(style(green, target.Name+" joins the crew. ("+itoa(len(p.party.Members))+" members)") + crlf)
}

func (w *World) showParty(p *Player) {
	if p.party == nil || len(p.party.Members) < 2 {
		p.send(style(dim, "You're running solo. Use GROUP <runner> to crew up.") + crlf)
		return
	}
	p.send(style(neon, "-- Your crew --") + crlf)
	for _, m := range p.party.Members {
		here := ""
		if m.RoomID == p.RoomID {
			here = style(dim, " (here)")
		}
		p.send("  " + style(green, m.Name) + style(dim, " (level "+itoa(m.Level)+")") + here + crlf)
	}
}

// leaveParty removes p from its crew, dissolving the crew if it drops below two.
func (w *World) leaveParty(p *Player) {
	if p.party == nil {
		p.send(style(dim, "You're not in a crew.") + crlf)
		return
	}
	pt := p.party
	pt.remove(p)
	p.send(style(green, "You leave the crew.") + crlf)
	pt.broadcast(style(dim, p.Name+" left the crew.") + crlf)
	w.dissolveIfTooSmall(pt)
}

// dropFromParty silently removes a (disconnecting) player from any crew.
func (w *World) dropFromParty(p *Player) {
	if p.party == nil {
		return
	}
	pt := p.party
	pt.remove(p)
	pt.broadcast(style(dim, p.Name+" dropped from the crew.") + crlf)
	w.dissolveIfTooSmall(pt)
}

func (w *World) dissolveIfTooSmall(pt *Party) {
	if len(pt.Members) < 2 {
		for _, m := range pt.Members {
			m.send(style(dim, "The crew dissolves.") + crlf)
			m.party = nil
		}
		pt.Members = nil
	}
}

// groupChat relays a message to the whole crew, wherever they are.
func (w *World) groupChat(p *Player, msg string) {
	msg = strings.TrimSpace(msg)
	if p.party == nil || len(p.party.Members) < 2 {
		p.send(style(dim, "You have no crew to radio.") + crlf)
		return
	}
	if msg == "" {
		p.send(style(dim, "Radio what?") + crlf)
		return
	}
	p.party.broadcast(style(hot, "[crew] "+p.Name+": ") + msg + crlf)
}

// awardXP grants kill XP, split across crew members in the same room (each gets
// an equal share, rounded up so grouping is never a penalty), and processes any
// resulting level-ups. Solo players get the full amount.
func (w *World) awardXP(killer *Player, xp int) {
	recipients := []*Player{killer}
	if killer.party != nil {
		recipients = recipients[:0]
		for _, m := range killer.party.Members {
			if m.RoomID == killer.RoomID {
				recipients = append(recipients, m)
			}
		}
	}
	share := xp
	if len(recipients) > 1 {
		share = (xp + len(recipients) - 1) / len(recipients) // ceil-divide, in players' favor
	}
	for _, r := range recipients {
		r.XP += share
		if r != killer {
			r.send(style(gold, "Crew share: +"+itoa(share)+" XP from "+killer.Name+"'s kill.") + crlf)
		}
		w.checkLevelUp(r)
	}
}
