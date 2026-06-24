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
		if len(inbox) == 0 {
			w.Line("  (your inbox is empty)")
		}
		for i, m := range inbox {
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
		w.Color(screen.Green)
		w.Print("\r\n[#] read  [C]ompose  [Q]uit: ")
		w.Reset()
		in, err := s.ReadLine()
		if err != nil {
			return err
		}
		in = strings.TrimSpace(in)
		switch {
		case in == "" || strings.EqualFold(in, "q"):
			return nil
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
	w.Print("\r\n[R]eply  [B]lock sender  re[P]ort  [Q]uit: ")
	w.Reset()
	key, err := s.ReadKey()
	if err != nil {
		return err
	}
	switch toLower(key) {
	case 'r':
		return sendMail(s, st, u, m.FromID, "re: "+m.Subject)
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
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Print("\r\n")
	w.Color(screen.Green)
	w.Print("To (handle): ")
	w.Reset()
	to, err := s.ReadLine()
	if err != nil {
		return err
	}
	recipient, err := st.Users().ByHandle(strings.TrimSpace(to))
	if err != nil {
		w.ColorLine(screen.Red, "no such user")
		return nil
	}
	return sendMail(s, st, u, recipient.ID, "")
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
