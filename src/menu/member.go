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
			{Key: 'F', Label: "File Library     (coming in Sprint 006)", Action: placeholder("file-library", "File Library")},
			{Key: 'D', Label: "Door Games       (coming in Sprint 007)", Action: placeholder("door-games", "Door Games")},
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
