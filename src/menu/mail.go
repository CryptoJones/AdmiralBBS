package menu

import (
	"fmt"
	"strconv"
	"strings"

	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// RunMail drives the private-mail subsystem (inbox, read, compose).
func RunMail(s *session.Session, st *store.Store, u *store.User) error {
	handles := newHandleCache(st)
	page := 0
	for {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		w.Clear()
		w.ColorLine(screen.Cyan, "Private Mail")
		w.ColorLine(screen.Cyan, "------------")
		inbox, err := st.Mail().Inbox(u.ID)
		if err != nil {
			return err
		}
		// Hide mail from users this member has blocked.
		if blocked, err := st.Blocks().BlockedSet(u.ID); err == nil && len(blocked) > 0 {
			kept := inbox[:0]
			for _, m := range inbox {
				if !blocked[m.FromID] {
					kept = append(kept, m)
				}
			}
			inbox = kept
		}
		lo, hi, pages := pageWindow(len(inbox), page)
		page = clampPage(page, pages)
		if len(inbox) == 0 {
			w.Line("  (your inbox is empty)")
		}
		for i := lo; i < hi; i++ {
			m := inbox[i]
			tag := "   "
			if m.ReadAt == nil {
				tag = "NEW"
			}
			w.Color(screen.Yellow)
			w.Printf("  %d) ", i+1)
			if m.ReadAt == nil {
				w.Color(screen.Magenta)
			} else {
				w.Color(screen.White)
			}
			w.Printf("[%s] ", tag)
			w.SafePrint(firstLine(m.Subject))
			w.Reset()
			w.Printf("  — from %s, %s\r\n", handles.handle(m.FromID), m.SentAt.Format("2006-01-02"))
		}
		pageFooter(w, page, pages)
		w.Color(screen.Green)
		w.Print("\r\n[#] read  [C]ompose  [Q]uit: ")
		w.Reset()
		in, err := s.ReadLine()
		if err != nil {
			return err
		}
		in = strings.TrimSpace(in)
		switch {
		case in == "" || strings.EqualFold(in, "q") || strings.EqualFold(in, "x"):
			return nil
		case in == ">":
			page++
		case in == "<":
			page--
		case strings.EqualFold(in, "c"):
			if err := composeMail(s, st, u, handles); err != nil {
				return err
			}
		default:
			if n, perr := strconv.Atoi(in); perr == nil && n >= 1 && n <= len(inbox) {
				if err := readMail(s, st, u, inbox[n-1].ID, handles); err != nil {
					return err
				}
			}
		}
	}
}

func readMail(s *session.Session, st *store.Store, u *store.User, id int64, handles *handleCache) error {
	m, err := st.Mail().Get(id, u.ID) // marks read
	if err != nil {
		return nil
	}
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.Color(screen.Cyan)
	w.Print("Subject: ")
	w.SafePrint(m.Subject)
	w.Print("\r\n")
	w.Reset()
	w.Printf("From: %s   %s\r\n\r\n", handles.handle(m.FromID), m.SentAt.Format("2006-01-02 15:04"))
	w.SafePrint(m.Body)
	w.Print("\r\n")
	w.Color(screen.Green)
	w.Print("\r\n[R]eply  [D]elete  [B]lock sender  re[P]ort  [Q]uit: ")
	w.Reset()
	key, err := s.ReadKey()
	if err != nil {
		return err
	}
	switch toLower(key) {
	case 'r':
		return sendMail(s, st, u, m.FromID, "re: "+m.Subject)
	case 'd':
		if err := st.Mail().Delete(m.ID, u.ID); err != nil {
			w.ColorLine(screen.Red, "could not delete")
		} else {
			s.Activity("delete-mail", "")
			w.ColorLine(screen.Cyan, "\r\nDeleted.")
			_, _ = s.ReadKey()
		}
	case 'b':
		if err := st.Blocks().Block(u.ID, m.FromID); err != nil {
			w.ColorLine(screen.Red, "could not block")
		} else {
			s.Activity("block-user", handles.handle(m.FromID))
			w.ColorLine(screen.Cyan, "\r\nBlocked. You won't see their mail or posts.")
			_, _ = s.ReadKey()
		}
	case 'p':
		reportUser(s, st, u, m.FromID, fmt.Sprintf("mail #%d", m.ID), handles)
	}
	return nil
}

// reportUser files an abuse report against target, routed to the SysOp queue.
func reportUser(s *session.Session, st *store.Store, u *store.User, targetID int64, context string, handles *handleCache) {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Color(screen.Green)
	w.Printf("\r\nReport %s to the SysOp. Describe the problem: ", handles.handle(targetID))
	w.Reset()
	note, err := s.ReadLine()
	if err != nil {
		return
	}
	if _, err := st.Reports().File(u.ID, targetID, context, note); err != nil {
		w.ColorLine(screen.Red, "could not file report")
		return
	}
	s.Activity("report-user", handles.handle(targetID))
	w.ColorLine(screen.Cyan, "Report filed. A SysOp will review it.")
	_, _ = s.ReadKey()
}

func composeMail(s *session.Session, st *store.Store, u *store.User, handles *handleCache) error {
	recipient, err := pickRecipient(s, st, u)
	if err != nil {
		return err
	}
	if recipient == nil {
		return nil // cancelled
	}
	return sendMail(s, st, u, recipient.ID, "")
}

// pickRecipient resolves the "To:" recipient. The caller can type a handle, or
// "?" to LOOK UP the member directory when they don't know the exact handle —
// they pick from a paged list instead. Returns (nil, nil) on cancel.
func pickRecipient(s *session.Session, st *store.Store, u *store.User) (*store.User, error) {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Print("\r\n")
	w.Color(screen.Green)
	w.Print("To (handle, or ? to look up): ")
	w.Reset()
	to, err := s.ReadLine()
	if err != nil {
		return nil, err
	}
	to = strings.TrimSpace(to)
	if to == "" {
		return nil, nil
	}
	if to == "?" {
		return lookupUser(s, st, u)
	}
	recipient, err := st.Users().ByHandle(to)
	if err != nil {
		w.ColorLine(screen.Red, "no such user — type ? to look up the directory")
		return nil, nil
	}
	return recipient, nil
}

// lookupUser shows a paged directory of approved members (excluding the caller)
// so someone who doesn't know a handle can pick a recipient by number.
func lookupUser(s *session.Session, st *store.Store, u *store.User) (*store.User, error) {
	all, err := st.Users().ListByStatus(store.StatusApproved)
	if err != nil {
		return nil, err
	}
	dir := all[:0]
	for _, m := range all {
		if m.ID != u.ID {
			dir = append(dir, m)
		}
	}
	if len(dir) == 0 {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		w.ColorLine(screen.Red, "no other members to message yet")
		_, _ = s.ReadKey()
		return nil, nil
	}
	page := 0
	for {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		lo, hi, pages := pageWindow(len(dir), page)
		page = clampPage(page, pages)
		w.Clear()
		w.ColorLine(screen.Cyan, "Member Directory")
		w.ColorLine(screen.Cyan, "----------------")
		for i := lo; i < hi; i++ {
			w.Color(screen.Yellow)
			w.Printf("  %d) ", i+1)
			w.Color(screen.White)
			w.SafePrint(dir[i].Handle)
			w.Reset()
			w.Print("\r\n")
		}
		pageFooter(w, page, pages)
		w.Color(screen.Green)
		w.Print("\r\n[#] pick  [>]/[<] page  [Q] cancel: ")
		w.Reset()
		in, err := s.ReadLine()
		if err != nil {
			return nil, err
		}
		in = strings.TrimSpace(in)
		switch {
		case in == "" || strings.EqualFold(in, "q") || strings.EqualFold(in, "x"):
			return nil, nil
		case in == ">":
			page++
		case in == "<":
			page--
		default:
			if n, perr := strconv.Atoi(in); perr == nil && n >= 1 && n <= len(dir) {
				return dir[n-1], nil
			}
		}
	}
}

func sendMail(s *session.Session, st *store.Store, u *store.User, toID int64, presetSubject string) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	subject := presetSubject
	if subject == "" {
		w.Color(screen.Green)
		w.Print("Subject: ")
		w.Reset()
		in, err := s.ReadLine()
		if err != nil {
			return err
		}
		subject = in
	}
	if strings.TrimSpace(subject) == "" {
		return nil
	}
	w.Line("Enter your message. End with a single '.' on its own line.")
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
	if _, err := st.Mail().Send(u.ID, toID, subject, strings.Join(lines, "\n")); err != nil {
		w.ColorLine(screen.Red, "could not send: "+err.Error())
		return nil
	}
	s.Activity("send-mail", "")
	w.ColorLine(screen.Cyan, "Mail sent.")
	return nil
}
