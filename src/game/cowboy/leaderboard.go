package cowboy

import "sort"

// sortSavedByRank orders characters for the leaderboard: highest level first,
// then most XP, then name for a stable tiebreak.
func sortSavedByRank(s []SavedPlayer) {
	sort.Slice(s, func(i, j int) bool {
		if s[i].Level != s[j].Level {
			return s[i].Level > s[j].Level
		}
		if s[i].XP != s[j].XP {
			return s[i].XP > s[j].XP
		}
		return s[i].Name < s[j].Name
	})
}

// leaderboard shows the top runners by saved progress. Online players' unsaved
// gains may lag their last save — classic BBS behavior.
func (w *World) leaderboard(p *Player) {
	top, err := w.store.Top(10)
	if err != nil {
		p.send(style(red, "The rankings are offline.") + crlf)
		return
	}
	p.send(crlf + style(neon, "== TOP CONSOLE COWBOYS ==") + crlf)
	if len(top) == 0 {
		p.send(style(dim, "  (no runners ranked yet)") + crlf)
		return
	}
	for i, sp := range top {
		class := sp.Class
		if class == "" {
			class = "runner"
		}
		p.send("  " + style(gold, itoa(i+1)+".") + " " + style(green, sp.Name) +
			style(dim, " — "+class+" L"+itoa(sp.Level)) + style(gold, "  €$"+itoa(sp.Eddies)) + crlf)
	}
}
