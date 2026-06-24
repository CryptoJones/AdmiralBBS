package menu

import (
	"admiralbbs/src/doors"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// Member builds the authenticated member's main menu. Subsystem entries are
// placeholders until their sprints land; the Profile entry (self-service SSH
// key management) is live now.
func Member(st *store.Store, u *store.User, artPath, auditPath string, doorOpts doors.Opts, node int, doorsData string, roster *session.Roster, sysopPass string) *Menu {
	items := []Item{
		{Key: 'M', Label: "Message Boards", Action: boardsAction(st, u)},
		{Key: 'E', Label: "Private Mail", Action: mailAction(st, u)},
		{Key: 'F', Label: "File Library", Action: filesAction(st, u)},
		{Key: 'D', Label: "Door Games", Action: doorsAction(st, u, doorOpts, node, doorsData)},
		{Key: 'W', Label: "Who's Online", Action: whosOnlineAction(roster)},
		{Key: 'K', Label: "My SSH Keys / Profile", Action: profileAction(st, u)},
	}
	if motd := st.Settings().MOTD(); motd != "" {
		items = append(items, Item{Key: 'O', Label: "Message of the Day (re-read)", Action: motdAction(st)})
	}
	if u.AccessLevel >= CoSysOpLevel {
		items = append(items, Item{Key: 'X', Label: "SysOp Control Panel", Action: sysopAction(st, u, auditPath, sysopPass)})
	}
	items = append(items, Item{Key: 'G', Label: "Goodbye / Logoff", Action: logoff})
	return &Menu{
		Title:    st.Settings().BBSName() + " :: Main Menu",
		Subtitle: st.Settings().Tagline(),
		ArtPath:  artPath,
		Items:    items,
	}
}

func motdAction(st *store.Store) Action {
	return func(s *session.Session) (Outcome, error) {
		if err := ShowMOTD(s, st.Settings().MOTD()); err != nil {
			return Logoff, err
		}
		return Continue, nil
	}
}

func sysopAction(st *store.Store, u *store.User, auditPath, sysopPass string) Action {
	return func(s *session.Session) (Outcome, error) {
		if err := RunSysOp(s, st, u, auditPath, sysopPass); err != nil {
			return Logoff, err
		}
		return Continue, nil
	}
}

func whosOnlineAction(roster *session.Roster) Action {
	return func(s *session.Session) (Outcome, error) {
		if err := RunWhosOnline(s, roster); err != nil {
			return Logoff, err
		}
		return Continue, nil
	}
}

func profileAction(st *store.Store, u *store.User) Action {
	return func(s *session.Session) (Outcome, error) {
		if err := RunProfile(s, st, u); err != nil {
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

func doorsAction(st *store.Store, u *store.User, doorOpts doors.Opts, node int, doorsData string) Action {
	return func(s *session.Session) (Outcome, error) {
		if err := RunDoors(s, st, u, doorOpts, node, doorsData); err != nil {
			return Logoff, err
		}
		return Continue, nil
	}
}
