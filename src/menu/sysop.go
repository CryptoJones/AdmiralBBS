package menu

import (
	"fmt"
	"strconv"
	"strings"

	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// CoSysOpLevel is the minimum access level for the control panel.
const CoSysOpLevel = 80

// RunSysOp is the SysOp Control Panel — SSH-only, access ≥80, server-side gated
// on every action. Membership approval, user management, content management, and
// the audit viewer.
func RunSysOp(s *session.Session, st *store.Store, u *store.User, auditPath string) error {
	if u.AccessLevel < CoSysOpLevel {
		return nil // defence in depth — the menu shouldn't have offered it
	}
	for {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		w.Clear()
		w.ColorLine(screen.Magenta, "SysOp Control Panel")
		w.ColorLine(screen.Magenta, "-------------------")
		pending, _ := st.Memberships().Pending()
		w.Printf("  [M] Membership queue (%d pending)\r\n", len(pending))
		w.Line("  [U] User management")
		w.Line("  [C] Create area / register door")
		w.Line("  [A] Audit log viewer")
		w.Line("  [B] IP banlist")
		w.Line("  [Q] Back to main menu")
		w.Color(screen.Green)
		w.Print("\r\nChoice: ")
		w.Reset()
		key, err := s.ReadKey()
		if err != nil {
			return err
		}
		switch toLower(key) {
		case 'm':
			if err := membershipQueue(s, st, u); err != nil {
				return err
			}
		case 'u':
			if err := userManagement(s, st); err != nil {
				return err
			}
		case 'c':
			if err := contentManagement(s, st); err != nil {
				return err
			}
		case 'a':
			if err := auditViewer(s, st, auditPath); err != nil {
				return err
			}
		case 'b':
			if err := banManagement(s, st, u); err != nil {
				return err
			}
		case 'q':
			return nil
		}
	}
}

func membershipQueue(s *session.Session, st *store.Store, sysop *store.User) error {
	handles := newHandleCache(st)
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Cyan, "Pending Membership Applications")
	pending, err := st.Memberships().Pending()
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		w.Line("  (none)")
		w.Print("\r\nPress any key...")
		_, e := s.ReadKey()
		return e
	}
	for i, m := range pending {
		w.Color(screen.Yellow)
		w.Printf("  %d) ", i+1)
		w.Color(screen.White)
		w.SafePrint(handles.handle(m.UserID))
		w.Reset()
		w.Print("  — ")
		w.SafePrint(firstLine(m.Note))
		w.Print("\r\n")
	}
	w.Color(screen.Green)
	w.Print("\r\nApprove/deny # (or [Q]): ")
	w.Reset()
	in, err := s.ReadLine()
	if err != nil {
		return err
	}
	n, perr := strconv.Atoi(strings.TrimSpace(in))
	if perr != nil || n < 1 || n > len(pending) {
		return nil
	}
	m := pending[n-1]
	w.Print("[A]pprove or [D]eny? ")
	dec, err := s.ReadKey()
	if err != nil {
		return err
	}
	switch toLower(dec) {
	case 'a':
		w.Color(screen.Green)
		w.Print("\r\nAccess level [50]: ")
		w.Reset()
		lvlIn, _ := s.ReadLine()
		level := 50
		if v, e := strconv.Atoi(strings.TrimSpace(lvlIn)); e == nil && v > 0 {
			level = v
		}
		if err := st.Users().Approve(m.UserID, level); err != nil {
			return err
		}
		if err := st.Memberships().Review(m.ID, sysop.ID, store.DecisionApproved, "approved via control panel"); err != nil {
			return err
		}
		tok, err := st.Tokens().Issue(m.UserID)
		if err != nil {
			return err
		}
		s.Activity("member-approve", handles.handle(m.UserID))
		w.Print("\r\n")
		w.ColorLine(screen.Cyan, "Approved. ONE-TIME TOKEN (relay out-of-band, e.g. PGP email):")
		w.Color(screen.Yellow)
		w.Print("    " + tok + "\r\n")
		w.Reset()
		w.Line("They set their password with this on first SSH login. It is shown once.")
	case 'd':
		if err := st.Users().SetStatus(m.UserID, store.StatusDenied, 0); err != nil {
			return err
		}
		_ = st.Memberships().Review(m.ID, sysop.ID, store.DecisionDenied, "denied via control panel")
		s.Activity("member-deny", handles.handle(m.UserID))
		w.ColorLine(screen.Red, "\r\nDenied.")
	default:
		return nil
	}
	w.Print("\r\nPress any key...")
	_, e := s.ReadKey()
	return e
}

func userManagement(s *session.Session, st *store.Store) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Cyan, "Users")
	users, err := st.Users().All()
	if err != nil {
		return err
	}
	for i, u := range users {
		w.Printf("  %d) %-16s %-9s lvl:%-3d %dmin/day\r\n", i+1, trunc(u.Handle, 16), u.Status, u.AccessLevel, u.DailyMinutes)
	}
	w.Color(screen.Green)
	w.Print("\r\nUser # to manage (or [Q]): ")
	w.Reset()
	in, err := s.ReadLine()
	if err != nil {
		return err
	}
	n, perr := strconv.Atoi(strings.TrimSpace(in))
	if perr != nil || n < 1 || n > len(users) {
		return nil
	}
	target := users[n-1]
	w.Printf("\r\n[L]evel  [S]uspend  [R]einstate  [T]ime budget  [P] clear password  [Q]: ")
	act, err := s.ReadKey()
	if err != nil {
		return err
	}
	switch toLower(act) {
	case 'l':
		w.Print("\r\nNew access level: ")
		v, _ := s.ReadLine()
		if lvl, e := strconv.Atoi(strings.TrimSpace(v)); e == nil {
			st.Users().SetStatus(target.ID, target.Status, lvl)
			w.ColorLine(screen.Cyan, "Updated.")
		}
	case 's':
		st.Users().SetStatus(target.ID, store.StatusSuspended, target.AccessLevel)
		w.ColorLine(screen.Red, "\r\nSuspended.")
	case 'r':
		st.Users().SetStatus(target.ID, store.StatusApproved, target.AccessLevel)
		w.ColorLine(screen.Cyan, "\r\nReinstated.")
	case 't':
		w.Print("\r\nDaily minutes: ")
		v, _ := s.ReadLine()
		if min, e := strconv.Atoi(strings.TrimSpace(v)); e == nil {
			st.Users().SetDailyMinutes(target.ID, min)
			w.ColorLine(screen.Cyan, "Updated.")
		}
	case 'p':
		st.Users().SetPassword(target.ID, "")
		w.ColorLine(screen.Cyan, "\r\nPassword cleared — they re-onboard on next login.")
	}
	w.Print("\r\nPress any key...")
	_, e := s.ReadKey()
	return e
}

func contentManagement(s *session.Session, st *store.Store) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.Print("[1] message area  [2] file area  [3] register door  [Q]: ")
	k, err := s.ReadKey()
	if err != nil {
		return err
	}
	switch k {
	case '1':
		name := prompt(s, w, "Area name: ")
		desc := prompt(s, w, "Description: ")
		if name != "" {
			st.MessageAreas().Create(name, desc, 0)
			w.ColorLine(screen.Cyan, "Created.")
		}
	case '2':
		name := prompt(s, w, "File area name: ")
		if name != "" {
			st.FileAreas().Create(name, 0)
			w.ColorLine(screen.Cyan, "Created.")
		}
	case '3':
		name := prompt(s, w, "Door name: ")
		if name == "" {
			break
		}
		w.Color(screen.Green)
		w.Print("\r\n[s]ubprocess door or [r]esident multiplayer server? ")
		w.Reset()
		k, _ := s.ReadKey()
		if toLower(k) == 'r' {
			addr := prompt(s, w, "Game address (host:port): ")
			if addr != "" {
				st.Doors().CreateResident(name, "tcp", addr, 0)
				w.ColorLine(screen.Cyan, "Registered resident (multiplayer) door.")
			}
		} else {
			cmd := prompt(s, w, "Command (path): ")
			if cmd != "" {
				st.Doors().Create(name, cmd, "door32.sys", 0)
				w.ColorLine(screen.Cyan, "Registered.")
			}
		}
	default:
		return nil
	}
	w.Print("\r\nPress any key...")
	_, e := s.ReadKey()
	return e
}

func auditViewer(s *session.Session, st *store.Store, auditPath string) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Cyan, "Audit Log — recent events")
	events, err := st.SessionLog().Recent(20)
	if err != nil {
		return err
	}
	for _, e := range events {
		w.Printf("  %s %-10s %-12s %-9s ", e.Time.Format("01-02 15:04"), trunc(e.Username, 10), trunc(e.SessionID, 12), e.Type)
		w.SafePrint(e.Action)
		w.Print("\r\n")
	}
	w.Print("\r\n")
	if auditPath != "" {
		n, verr := st.VerifyAuditChain(auditPath)
		if verr != nil {
			w.ColorLine(screen.Red, fmt.Sprintf("CHAIN VERIFY FAILED — possible tampering: %v", verr))
		} else {
			w.ColorLine(screen.Green, fmt.Sprintf("Authoritative JSONL chain verified intact: %d events.", n))
		}
	}
	w.Print("\r\nPress any key...")
	_, e := s.ReadKey()
	return e
}

func banManagement(s *session.Session, st *store.Store, sysop *store.User) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Magenta, "IP Banlist")
	w.ColorLine(screen.Magenta, "----------")
	active, err := st.Bans().Active()
	if err != nil {
		return err
	}
	if len(active) == 0 {
		w.Line("  (no active bans)")
	}
	for i, b := range active {
		w.Printf("  %d) %-20s ", i+1, b.Pattern)
		w.SafePrint(firstLine(b.Reason))
		w.Print("\r\n")
	}
	w.Color(screen.Green)
	w.Print("\r\n[A]dd ban  [L]ift ban  [Q]uit: ")
	w.Reset()
	k, err := s.ReadKey()
	if err != nil {
		return err
	}
	switch toLower(k) {
	case 'a':
		pattern := prompt(s, w, "IP or CIDR (e.g. 203.0.113.7 or 203.0.113.0/24): ")
		if pattern == "" {
			break
		}
		reason := prompt(s, w, "Reason: ")
		if _, err := st.Bans().Add(pattern, reason, sysop.ID); err != nil {
			w.ColorLine(screen.Red, "  [x] "+err.Error())
		} else {
			w.ColorLine(screen.Cyan, "  [ok] banned — new connections from there are dropped at accept")
			s.Activity("ip-ban", pattern)
		}
	case 'l':
		if len(active) == 0 {
			break
		}
		in := prompt(s, w, "Number to lift (or blank to cancel): ")
		n, perr := strconv.Atoi(strings.TrimSpace(in))
		if perr != nil || n < 1 || n > len(active) {
			break
		}
		if err := st.Bans().Lift(active[n-1].ID); err != nil {
			w.ColorLine(screen.Red, "  [x] could not lift")
		} else {
			w.ColorLine(screen.Cyan, "  [ok] ban lifted")
			s.Activity("ip-unban", active[n-1].Pattern)
		}
	default:
		return nil
	}
	w.Print("\r\nPress any key...")
	_, e := s.ReadKey()
	return e
}

func prompt(s *session.Session, w *screen.Writer, label string) string {
	w.Color(screen.Green)
	w.Print("\r\n" + label)
	w.Reset()
	v, _ := s.ReadLine()
	return strings.TrimSpace(v)
}

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
