package menu

import (
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// Member builds the authenticated member's main menu. Subsystem entries are
// placeholders until their sprints land; the Profile entry (self-service SSH
// key management) is live now.
func Member(st *store.Store, u *store.User, artPath string) *Menu {
	return &Menu{
		Title:   "AdmiralBBS :: Main Menu",
		ArtPath: artPath,
		Items: []Item{
			{Key: 'M', Label: "Message Boards", Action: boardsAction(st, u)},
			{Key: 'E', Label: "Private Mail", Action: mailAction(st, u)},
			{Key: 'F', Label: "File Library", Action: filesAction(st, u)},
			{Key: 'D', Label: "Door Games", Action: doorsAction(st, u)},
			{Key: 'K', Label: "My SSH Keys / Profile", Action: profileAction(st, u)},
			{Key: 'G', Label: "Goodbye / Logoff", Action: logoff},
		},
	}
}

func profileAction(st *store.Store, u *store.User) Action {
	return func(s *session.Session) (Outcome, error) {
		if err := RunProfile(s, u, st.Keys()); err != nil {
			return Logoff, err
		}
		return Continue, nil
	}
}

func boardsAction(st *store.Store, u *store.User) Action {
	return func(s *session.Session) (Outcome, error) {
		if err := RunBoards(s, st, u); err != nil {
			return Logoff, err
		}
		return Continue, nil
	}
}

func mailAction(st *store.Store, u *store.User) Action {
	return func(s *session.Session) (Outcome, error) {
		if err := RunMail(s, st, u); err != nil {
			return Logoff, err
		}
		return Continue, nil
	}
}

func filesAction(st *store.Store, u *store.User) Action {
	return func(s *session.Session) (Outcome, error) {
		if err := RunFiles(s, st, u); err != nil {
			return Logoff, err
		}
		return Continue, nil
	}
}

func doorsAction(st *store.Store, u *store.User) Action {
	return func(s *session.Session) (Outcome, error) {
		if err := RunDoors(s, st, u); err != nil {
			return Logoff, err
		}
		return Continue, nil
	}
}
