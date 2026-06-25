package menu

import (
	"crypto/subtle"
	"fmt"
	"strconv"
	"strings"

	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// CoSysOpLevel is the minimum access level for the control panel.
const CoSysOpLevel = store.SysOpLevel

// sysopPassOK constant-time compares the entered panel password to the
// configured shared secret.
func sysopPassOK(provided, configured string) bool {
	return subtle.ConstantTimeCompare([]byte(provided), []byte(configured)) == 1
}

// RunSysOp is the SysOp Control Panel — SSH-only, access ≥80, server-side gated
// on every action. If sysopPass is non-empty it is a SHARED step-up secret that
// must be entered before the panel opens — so an unattended logged-in SysOp
// session can't be used to change BBS settings by someone who doesn't know it.
func RunSysOp(s *session.Session, st *store.Store, u *store.User, auditPath, sysopPass string, installer *DoorInstaller) error {
	if u.AccessLevel < CoSysOpLevel {
		return nil // defence in depth — the menu shouldn't have offered it
	}
	if sysopPass != "" {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		w.Color(screen.Green)
		w.Print("\r\nSysOp panel password: ")
		w.Reset()
		entered, err := s.ReadPassword()
		if err != nil {
			return err
		}
		if !sysopPassOK(entered, sysopPass) {
			s.Activity("sysop-gate-failed", "")
			w.ColorLine(screen.Red, "Denied.")
			_, _ = s.ReadKey()
			return nil
		}
		s.Activity("sysop-gate-ok", "")
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
		if installer != nil {
			w.Line("  [I] Install door from release URL")
		}
		w.Line("  [A] Audit log viewer")
		w.Line("  [B] IP banlist")
		openReports, _ := st.Reports().OpenCount()
		w.Printf("  [R] Abuse reports (%d open)\r\n", openReports)
		w.Line("  [S] Branding & MOTD")
		w.Line("  [T] Session timeouts")
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
		case 'i':
			if installer != nil {
				if err := installDoorScreen(s, st, installer); err != nil {
					return err
				}
			}
		case 'a':
			if err := auditViewer(s, st, auditPath); err != nil {
				return err
			}
		case 'b':
			if err := banManagement(s, st, u); err != nil {
				return err
			}
		case 'r':
			if err := reportsQueue(s, st, u); err != nil {
				return err
			}
		case 's':
			if err := editSettings(s, st); err != nil {
				return err
			}
		case 't':
			if err := editTimeouts(s, st); err != nil {
				return err
			}
		case 'q':
			return nil
		}
	}
}

// installDoorScreen lets a SysOp add a resident door by pointing the BBS at a
// forge release URL. The BBS downloads the binary matching ITS OWN OS/arch, runs
// it supervised, and registers it. Running a downloaded binary is a trust
// decision, so this is SysOp-gated.
func installDoorScreen(s *session.Session, st *store.Store, installer *DoorInstaller) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Magenta, "Install door from release URL")
	w.ColorLine(screen.Magenta, "-----------------------------")
	w.Line("The BBS downloads the binary built for THIS server's OS/arch, runs it")
	w.Line("supervised, and registers it. Only install doors you trust.")
	w.Print("\r\n")

	if list, _ := st.InstalledDoors().List(); len(list) > 0 {
		w.ColorLine(screen.Cyan, "Already installed:")
		for _, d := range list {
			w.Printf("  %s (%s) @ %s\r\n", d.Name, d.Version, d.Address)
		}
		w.Print("\r\n")
	}

	w.Color(screen.Green)
	w.Print("Door name (blank to cancel): ")
	w.Reset()
	name, err := s.ReadLine()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	w.Color(screen.Green)
	w.Print("Release URL (forge releases JSON, e.g. .../releases/latest): ")
	w.Reset()
	url, err := s.ReadLine()
	if err != nil {
		return err
	}
	url = strings.TrimSpace(url)
	if url == "" {
		w.ColorLine(screen.Red, "No URL — cancelled.")
		_, _ = s.ReadKey()
		return nil
	}

	w.Color(screen.Green)
	w.Print("Min access level [0]: ")
	w.Reset()
	lvlStr, err := s.ReadLine()
	if err != nil {
		return err
	}
	minLevel := 0
	if v, e := strconv.Atoi(strings.TrimSpace(lvlStr)); e == nil {
		minLevel = v
	}

	w.ColorLine(screen.Cyan, "Fetching + downloading — this can take a moment...")
	s.Activity("door-install", name+" <- "+url)
	ver, ierr := installer.Install(name, url, minLevel)
	if ierr != nil {
		w.ColorLine(screen.Red, "Install failed: "+ierr.Error())
		_, _ = s.ReadKey()
		return nil
	}
	w.ColorLine(screen.Green, "Installed + launched: "+name+" "+ver+" — callers can play it now.")
	_, _ = s.ReadKey()
	return nil
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

// auditPageSize is how many audit events fill one screen of the paged viewer.
const auditPageSize = 15

func auditViewer(s *session.Session, st *store.Store, auditPath string) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)

	total, err := st.SessionLog().Count()
	if err != nil {
		return err
	}
	totalPages := (total + auditPageSize - 1) / auditPageSize
	if totalPages < 1 {
		totalPages = 1
	}

	// One-time summaries (expensive / stable) computed before the nav loop so
	// paging is snappy: the JSONL chain verify reads the whole trail, and the
	// anomaly list is a fixed recent snapshot.
	var chainMsg string
	var chainColor = screen.Green
	if auditPath != "" {
		if n, verr := st.VerifyAuditChain(auditPath); verr != nil {
			chainColor = screen.Red
			chainMsg = fmt.Sprintf("CHAIN VERIFY FAILED — possible tampering: %v", verr)
		} else {
			chainMsg = fmt.Sprintf("Authoritative JSONL chain verified intact: %d events.", n)
		}
	}
	anomalies, _ := st.Anomalies().Recent(5)
	handles := newHandleCache(st)

	page := 0 // 0 = newest page; higher = older
	for {
		events, err := st.SessionLog().Page(auditPageSize, page*auditPageSize)
		if err != nil {
			return err
		}
		w.Clear()
		first := page*auditPageSize + 1
		last := page*auditPageSize + len(events)
		w.ColorLine(screen.Cyan, fmt.Sprintf("Audit Log — page %d/%d  (events %d-%d of %d, newest first)",
			page+1, totalPages, first, last, total))
		for _, e := range events {
			w.Printf("  %s %-10s %-12s %-9s ", e.Time.Format("01-02 15:04"), trunc(e.Username, 10), trunc(e.SessionID, 12), e.Type)
			w.SafePrint(e.Action)
			w.Print("\r\n")
		}
		if len(events) == 0 {
			w.Line("  (no events on this page)")
		}
		// Rapid-IP-change ("impossible travel") flags — visibility only.
		if len(anomalies) > 0 {
			w.Print("\r\n")
			w.ColorLine(screen.Yellow, "Rapid IP changes (review — not auto-blocked):")
			for _, a := range anomalies {
				w.Printf("  %s %-10s %s -> %s after %ds\r\n",
					a.At.Format("01-02 15:04"), trunc(handles.handle(a.UserID), 10),
					a.PrevIP, a.NewIP, a.GapSeconds)
			}
		}
		w.Print("\r\n")
		if chainMsg != "" {
			w.ColorLine(chainColor, chainMsg)
		}
		w.Print("\r\n")
		w.ColorLine(screen.White, "[N]ext page  [P]rev  [F] +10 pages  [R] -10 pages  [Q] exit")
		w.Color(screen.Green)
		w.Print("Choice: ")
		w.Reset()

		k, e := s.ReadKey()
		if e != nil {
			return e
		}
		switch k {
		case 'n', 'N', '\r', '\n', ' ':
			page++
		case 'p', 'P':
			page--
		case 'f', 'F', '>', '+':
			page += 10
		case 'r', 'R', '<', '-':
			page -= 10
		case 'q', 'Q':
			return nil
		}
		if page > totalPages-1 {
			page = totalPages - 1
		}
		if page < 0 {
			page = 0
		}
	}
}

func reportsQueue(s *session.Session, st *store.Store, sysop *store.User) error {
	handles := newHandleCache(st)
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Magenta, "Abuse Reports — open")
	w.ColorLine(screen.Magenta, "--------------------")
	open, err := st.Reports().Open()
	if err != nil {
		return err
	}
	if len(open) == 0 {
		w.Line("  (no open reports)")
		w.Print("\r\nPress any key...")
		_, e := s.ReadKey()
		return e
	}
	for i, r := range open {
		w.Color(screen.Yellow)
		w.Printf("  %d) ", i+1)
		w.Color(screen.White)
		w.Printf("%s reported ", handles.handle(r.ReporterID))
		w.SafePrint(handles.handle(r.TargetID))
		w.Reset()
		w.Printf("  [%s] ", r.Context)
		w.SafePrint(firstLine(r.Note))
		w.Print("\r\n")
	}
	w.Color(screen.Green)
	w.Print("\r\nReport # to act on (or [Q]): ")
	w.Reset()
	in, err := s.ReadLine()
	if err != nil {
		return err
	}
	n, perr := strconv.Atoi(strings.TrimSpace(in))
	if perr != nil || n < 1 || n > len(open) {
		return nil
	}
	r := open[n-1]
	w.Printf("\r\n[S]uspend %s  [R]esolve (no action)  [Q]: ", handles.handle(r.TargetID))
	act, err := s.ReadKey()
	if err != nil {
		return err
	}
	switch toLower(act) {
	case 's':
		if target, e := st.Users().ByID(r.TargetID); e == nil {
			st.Users().SetStatus(target.ID, store.StatusSuspended, target.AccessLevel)
			w.ColorLine(screen.Red, "\r\nUser suspended.")
		}
		_ = st.Reports().Resolve(r.ID, sysop.ID)
		s.Activity("report-resolve", "suspend "+handles.handle(r.TargetID))
	case 'r':
		_ = st.Reports().Resolve(r.ID, sysop.ID)
		w.ColorLine(screen.Cyan, "\r\nResolved.")
		s.Activity("report-resolve", "noaction "+handles.handle(r.TargetID))
	default:
		return nil
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

// editSettings lets a SysOp customize the BBS name, tagline, and message of the
// day (shown to callers before the menu).
// editTimeouts lets a SysOp change the idle-disconnect timeout and the default
// daily time budget at runtime. Both persist in the settings table and apply to
// NEW sessions without a restart (the CLI flags only seed the defaults).
func editTimeouts(s *session.Session, st *store.Store) error {
	set := st.Settings()
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Magenta, "Session Timeouts")
	w.ColorLine(screen.Magenta, "----------------")
	w.Printf("  Idle disconnect: %d min\r\n", set.IdleMinutes(10))
	w.Printf("  Daily budget:    %d min  (members without a per-user override)\r\n", set.DailyMinutes(60))
	w.Color(screen.Green)
	w.Print("\r\nEdit [I]dle  [D]aily  [Q]uit: ")
	w.Reset()
	k, err := s.ReadKey()
	if err != nil {
		return err
	}
	switch toLower(k) {
	case 'i':
		if n, ok := promptMinutes(s, w, "New idle timeout in minutes (blank to cancel): "); ok {
			_ = st.Settings().Set("idle_minutes", strconv.Itoa(n))
			w.ColorLine(screen.Cyan, "Updated — applies to new sessions.")
			_, _ = s.ReadKey()
		}
	case 'd':
		if n, ok := promptMinutes(s, w, "New daily budget in minutes (blank to cancel): "); ok {
			_ = st.Settings().Set("daily_minutes", strconv.Itoa(n))
			w.ColorLine(screen.Cyan, "Updated — applies to new sessions.")
			_, _ = s.ReadKey()
		}
	}
	return nil
}

// promptMinutes prompts for a positive integer; (n, true) on a valid entry,
// (0, false) on blank/cancel or invalid.
func promptMinutes(s *session.Session, w *screen.Writer, label string) (int, bool) {
	v := strings.TrimSpace(prompt(s, w, label))
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		w.ColorLine(screen.Red, "Enter a positive number of minutes.")
		_, _ = s.ReadKey()
		return 0, false
	}
	return n, true
}

func editSettings(s *session.Session, st *store.Store) error {
	set := st.Settings()
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Magenta, "Branding & MOTD")
	w.ColorLine(screen.Magenta, "---------------")
	w.Printf("  Name:    %s\r\n", set.BBSName())
	w.Printf("  Tagline: %s\r\n", set.Tagline())
	motd := set.MOTD()
	if motd == "" {
		motd = "(none)"
	}
	w.Print("  MOTD:    ")
	w.SafePrint(firstLine(motd))
	w.Print("\r\n")
	w.Color(screen.Green)
	w.Print("\r\nEdit [N]ame  [T]agline  [M]OTD  [Q]uit: ")
	w.Reset()
	k, err := s.ReadKey()
	if err != nil {
		return err
	}
	switch toLower(k) {
	case 'n':
		if v := prompt(s, w, "New BBS name (blank to cancel): "); v != "" {
			st.Settings().Set("bbs_name", v)
			w.ColorLine(screen.Cyan, "Updated.")
		}
	case 't':
		if v := prompt(s, w, "New tagline (blank to cancel): "); v != "" {
			st.Settings().Set("tagline", v)
			w.ColorLine(screen.Cyan, "Updated.")
		}
	case 'm':
		w.Color(screen.Green)
		w.Print("\r\nEnter the MOTD. End with a single '.' on its own line; a lone '.' clears it.\r\n")
		w.Reset()
		var lines []string
		for {
			line, err := s.ReadLine()
			if err != nil {
				return err
			}
			if line == "." {
				break
			}
			lines = append(lines, line)
		}
		st.Settings().Set("motd", strings.Join(lines, "\n"))
		w.ColorLine(screen.Cyan, "MOTD updated.")
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
