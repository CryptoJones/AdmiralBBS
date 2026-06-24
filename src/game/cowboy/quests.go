package cowboy

import (
	"strconv"
	"strings"
)

// Quest is a broker bounty: kill Count of a target mob, then CLAIM at any broker
// (vendor room) for the reward. Bounties are repeatable.
type Quest struct {
	ID       string
	Name     string
	Desc     string
	Target   string // mob template ID
	Count    int
	XP       int
	Eddies   int
	MinLevel int
}

var quests = []Quest{
	{ID: "clear_alley", Name: "Clear the Back Alley", Target: "ganger", Count: 3, XP: 120, Eddies: 80, MinLevel: 1,
		Desc: "Gangers are taxing the block. Drop 3 street gangers."},
	{ID: "corp_sabotage", Name: "Corporate Sabotage", Target: "drone", Count: 2, XP: 200, Eddies: 150, MinLevel: 2,
		Desc: "A rival corp wants deniable chaos. Wreck 2 security drones in Corporate Plaza."},
	{ID: "break_ice", Name: "Break the Ice", Target: "white_ice", Count: 3, XP: 400, Eddies: 250, MinLevel: 3,
		Desc: "Prove you can run. Destroy 3 White ICE sentinels in the Net."},
	{ID: "ghost_machine", Name: "Ghost in the Machine", Target: "rogue_ai", Count: 1, XP: 1000, Eddies: 800, MinLevel: 5,
		Desc: "The Rogue AI in the Deep Net must die. End it."},
}

func questByID(id string) (Quest, bool) {
	for _, q := range quests {
		if q.ID == id {
			return q, true
		}
	}
	return Quest{}, false
}

// quests command: show the bounty board (at a broker) plus your active progress.
func (w *World) showQuests(p *Player) {
	atFixer := w.atVendor(p)
	if atFixer {
		p.send(crlf + style(neon, "== FIXER BOUNTY BOARD ==  ") + style(dim, "(ACCEPT <#>)") + crlf)
		for i, q := range quests {
			lvl := ""
			if p.Level < q.MinLevel {
				lvl = style(red, "  [needs level "+itoa(q.MinLevel)+"]")
			}
			p.send("  " + style(gold, itoa(i+1)+")") + " " + style(green, q.Name) +
				style(dim, " — "+q.Desc) + " " + style(gold, "(+"+itoa(q.XP)+"xp, €$"+itoa(q.Eddies)+")") + lvl + crlf)
		}
	}
	if len(p.Quests) == 0 {
		p.send(style(dim, "You have no active bounties.") + crlf)
		return
	}
	p.send(style(neon, "-- Active bounties --") + crlf)
	for id, got := range p.Quests {
		q, ok := questByID(id)
		if !ok {
			continue
		}
		state := itoa(got) + "/" + itoa(q.Count)
		if got >= q.Count {
			state = style(gold, "READY — CLAIM at a broker")
		}
		p.send("  " + style(green, q.Name) + style(dim, " ["+q.Target+"] ") + state + crlf)
	}
}

// accept takes a bounty (only at a broker).
func (w *World) accept(p *Player, arg string) {
	if !w.atVendor(p) {
		p.send(style(dim, "Find a broker (a vendor room) to take bounties.") + crlf)
		return
	}
	n, err := strconv.Atoi(strings.TrimSpace(arg))
	if err != nil || n < 1 || n > len(quests) {
		p.send(style(dim, "Accept which? See QUESTS for the numbered board.") + crlf)
		return
	}
	q := quests[n-1]
	if p.Level < q.MinLevel {
		p.send(style(red, "You need level "+itoa(q.MinLevel)+" for that job.") + crlf)
		return
	}
	if _, active := p.Quests[q.ID]; active {
		p.send(style(dim, "You're already on that bounty.") + crlf)
		return
	}
	p.Quests[q.ID] = 0
	p.send(style(green, "Bounty accepted: ") + q.Name + style(dim, " — "+q.Desc) + crlf)
}

// claim turns in any completed bounties (at a broker) for rewards.
func (w *World) claim(p *Player) {
	if !w.atVendor(p) {
		p.send(style(dim, "Return to a broker (a vendor room) to claim bounties.") + crlf)
		return
	}
	claimed := 0
	for id, got := range p.Quests {
		q, ok := questByID(id)
		if !ok || got < q.Count {
			continue
		}
		delete(p.Quests, id)
		p.XP += q.XP
		p.Eddies += q.Eddies
		claimed++
		p.send(style(gold, "*** Bounty paid: "+q.Name+" — +"+itoa(q.XP)+"xp, €$"+itoa(q.Eddies)+" ***") + crlf)
	}
	if claimed == 0 {
		p.send(style(dim, "No completed bounties to claim.") + crlf)
		return
	}
	w.checkLevelUp(p)
}

// creditQuestKill advances any active bounty whose target matches the slain mob.
func (w *World) creditQuestKill(p *Player, mobID string) {
	for id, got := range p.Quests {
		q, ok := questByID(id)
		if !ok || q.Target != mobID || got >= q.Count {
			continue
		}
		p.Quests[id] = got + 1
		if p.Quests[id] >= q.Count {
			p.send(style(gold, "Bounty objective complete: "+q.Name+" — CLAIM at a broker.") + crlf)
		} else {
			p.send(style(dim, "Bounty progress: "+q.Name+" "+itoa(p.Quests[id])+"/"+itoa(q.Count)) + crlf)
		}
	}
}
