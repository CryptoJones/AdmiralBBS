package menu

import (
	"fmt"
	"time"

	"admiralbbs/src/screen"
	"admiralbbs/src/session"
)

// RunWhosOnline shows the live roster of connected callers — node, handle,
// transport, and how long they've been on. A nil roster degrades gracefully.
func RunWhosOnline(s *session.Session, roster *session.Roster) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Cyan, "Who's Online")
	w.ColorLine(screen.Cyan, "------------")
	var list []session.Online
	if roster != nil {
		list = roster.List()
	}
	if len(list) == 0 {
		w.Line("  (nobody — you must be a ghost)")
	}
	for _, o := range list {
		w.Color(screen.Yellow)
		w.Printf("  Node %-3d ", o.Node)
		w.Color(screen.White)
		w.SafePrint(fmt.Sprintf("%-16s", o.Handle))
		w.Reset()
		w.Printf("  %-6s  on %s\r\n", o.Transport, since(o.Since))
	}
	w.Color(screen.Green)
	w.Print("\r\nPress any key...")
	w.Reset()
	_, err := s.ReadKey()
	return err
}

// since renders a compact elapsed string from t to now.
func since(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if h := int(d.Hours()); h > 0 {
		return fmt.Sprintf("%dh %dm", h, int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}
