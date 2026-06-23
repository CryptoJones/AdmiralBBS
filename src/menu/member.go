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
			{Key: 'M', Label: "Message Boards   (coming in Sprint 004)", Action: placeholder("message-boards", "Message Boards")},
			{Key: 'E', Label: "Private Mail     (coming in Sprint 005)", Action: placeholder("private-mail", "Private Mail")},
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
