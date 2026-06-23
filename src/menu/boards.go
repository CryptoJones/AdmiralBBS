package menu

import (
	"strconv"
	"strings"

	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// RunBoards drives the message-board subsystem for a logged-in member.
func RunBoards(s *session.Session, st *store.Store, u *store.User) error {
	handles := newHandleCache(st)
	for {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		w.Clear()
		w.ColorLine(screen.Cyan, "Message Boards")
		w.ColorLine(screen.Cyan, "--------------")
		areas, err := st.MessageAreas().Visible(u.AccessLevel)
		if err != nil {
			return err
		}
		for i, a := range areas {
			w.Color(screen.Yellow)
			w.Printf("  %d) ", i+1)
			w.Color(screen.White)
			w.SafePrint(a.Name)
			w.Reset()
			w.Print(" — ")
			w.SafePrint(a.Description)
			w.Print("\r\n")
		}
		w.Color(screen.Green)
		w.Print("\r\nArea # (or [Q]uit): ")
		w.Reset()
		in, err := s.ReadLine()
		if err != nil {
			return err
		}
		in = strings.TrimSpace(in)
		if in == "" || strings.EqualFold(in, "q") {
			return nil
		}
		n, perr := strconv.Atoi(in)
		if perr != nil || n < 1 || n > len(areas) {
			continue
		}
		if err := browseArea(s, st, u, areas[n-1], handles); err != nil {
			return err
		}
	}
}

func browseArea(s *session.Session, st *store.Store, u *store.User, area *store.MessageArea, handles *handleCache) error {
	for {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		w.Clear()
		w.Color(screen.Cyan)
		w.Print("Board: ")
		w.SafePrint(area.Name)
		w.Print("\r\n")
		w.Reset()
		msgs, err := st.Messages().Thread(area.ID)
		if err != nil {
			return err
		}
		if len(msgs) == 0 {
			w.Line("  (no messages yet — be the first)")
		}
		for i, m := range msgs {
			w.Color(screen.Yellow)
			w.Printf("  %d) ", i+1)
			w.Color(screen.White)
			w.SafePrint(firstLine(m.Subject))
			w.Reset()
			w.Printf("  — %s, %s\r\n", handles.handle(m.AuthorID), m.PostedAt.Format("2006-01-02"))
		}
		w.Color(screen.Green)
		w.Print("\r\n[#] read  [P]ost  [Q]uit: ")
		w.Reset()
		in, err := s.ReadLine()
		if err != nil {
			return err
		}
		in = strings.TrimSpace(in)
		switch {
		case in == "" || strings.EqualFold(in, "q"):
			return nil
		case strings.EqualFold(in, "p"):
			if err := compose(s, st, u, area.ID, nil); err != nil {
				return err
			}
		default:
			if n, perr := strconv.Atoi(in); perr == nil && n >= 1 && n <= len(msgs) {
				if err := readMessage(s, st, u, msgs[n-1], handles); err != nil {
					return err
				}
			}
		}
	}
}

func readMessage(s *session.Session, st *store.Store, u *store.User, m *store.Message, handles *handleCache) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.Color(screen.Cyan)
	w.Print("Subject: ")
	w.SafePrint(m.Subject)
	w.Print("\r\n")
	w.Reset()
	w.Printf("From: %s   %s\r\n\r\n", handles.handle(m.AuthorID), m.PostedAt.Format("2006-01-02 15:04"))
	w.SafePrint(m.Body)
	w.Print("\r\n")

	replies, err := st.Messages().Replies(m.ID)
	if err != nil {
		return err
	}
	for _, rep := range replies {
		w.Color(screen.Blue)
		w.Printf("\r\n  ┌ reply from %s, %s\r\n", handles.handle(rep.AuthorID), rep.PostedAt.Format("2006-01-02"))
		w.Reset()
		w.Print("  ")
		w.SafePrint(strings.ReplaceAll(rep.Body, "\n", "\n  "))
		w.Print("\r\n")
	}

	w.Color(screen.Green)
	w.Print("\r\n[R]eply  [Q]uit: ")
	w.Reset()
	key, err := s.ReadKey()
	if err != nil {
		return err
	}
	if toLower(key) == 'r' {
		return compose(s, st, u, m.AreaID, &m.ID)
	}
	return nil
}

func compose(s *session.Session, st *store.Store, u *store.User, areaID int64, parentID *int64) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Print("\r\n")
	w.Color(screen.Green)
	w.Print("Subject: ")
	w.Reset()
	subject, err := s.ReadLine()
	if err != nil {
		return err
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
	if _, err := st.Messages().Post(areaID, u.ID, parentID, subject, strings.Join(lines, "\n")); err != nil {
		w.ColorLine(screen.Red, "could not post: "+err.Error())
		return nil
	}
	s.Activity("post-message", "")
	w.ColorLine(screen.Cyan, "Posted.")
	return nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// handleCache resolves author IDs to handles, caching within a session.
type handleCache struct {
	st *store.Store
	m  map[int64]string
}

func newHandleCache(st *store.Store) *handleCache { return &handleCache{st: st, m: map[int64]string{}} }

func (c *handleCache) handle(id int64) string {
	if h, ok := c.m[id]; ok {
		return h
	}
	h := "unknown"
	if u, err := c.st.Users().ByID(id); err == nil {
		h = u.Handle
	}
	c.m[id] = h
	return h
}
