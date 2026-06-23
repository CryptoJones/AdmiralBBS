package menu

import (
	"admiralbbs/src/screen"
	"admiralbbs/src/session"
)

// Demo builds the Sprint 002 main menu: enough to prove the spine end-to-end.
// Real subsystems (boards, mail, files, doors) replace the placeholders in
// later sprints. artPath points at the CP437 welcome banner (may be "").
func Demo(artPath string) *Menu {
	return &Menu{
		Title:   "AdmiralBBS :: Main Menu",
		ArtPath: artPath,
		Items: []Item{
			{Key: 'M', Label: "Message Boards   (coming in Sprint 004)", Action: placeholder("message-boards", "Message Boards")},
			{Key: 'E', Label: "Private Mail     (coming in Sprint 005)", Action: placeholder("private-mail", "Private Mail")},
			{Key: 'F', Label: "File Library     (coming in Sprint 006)", Action: placeholder("file-library", "File Library")},
			{Key: 'D', Label: "Door Games       (coming in Sprint 007)", Action: placeholder("door-games", "Door Games")},
			{Key: 'G', Label: "Goodbye / Logoff", Action: logoff},
		},
	}
}

// placeholder records the activity in the audit trail and shows a stub screen.
func placeholder(action, title string) Action {
	return func(s *session.Session) (Outcome, error) {
		s.Activity(action, "selected from main menu")
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		w.Clear()
		w.ColorLine(screen.Magenta, title)
		w.Print("\r\n")
		w.Line("This subsystem is not built yet. The spine works, though —")
		w.Line("you connected, your terminal was detected, this menu routed")
		w.Line("your keypress, and this visit was written to the audit log.")
		w.Print("\r\nPress any key to return to the menu...")
		if _, err := s.ReadKey(); err != nil {
			return Logoff, err
		}
		return Continue, nil
	}
}

func logoff(s *session.Session) (Outcome, error) {
	s.Activity("logoff", "chose Goodbye")
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Cyan, "Thanks for calling AdmiralBBS. NO CARRIER")
	return Logoff, nil
}
