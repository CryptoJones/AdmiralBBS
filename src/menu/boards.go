package menu

import (
	"fmt"
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
	newestFirst := false
	page := 0
	var listed []*store.Message // non-nil => an active search/filter result
	var listedHeader string
	for {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)

		msgs := listed
		header := listedHeader
		if msgs == nil {
			var err error
			msgs, err = st.Messages().ThreadSorted(area.ID, newestFirst)
			if err != nil {
				return err
			}
		}
		msgs = hideBlocked(st, u.ID, msgs)
		lo, hi, pages := pageWindow(len(msgs), page)
		page = clampPage(page, pages)

		w.Clear()
		w.Color(screen.Cyan)
		w.Print("Board: ")
		w.SafePrint(area.Name)
		ord := "oldest first"
		if newestFirst {
			ord = "newest first"
		}
		w.Printf("   (%s)\r\n", ord)
		w.Reset()
		if header != "" {
			w.ColorLine(screen.Blue, header)
		}
		if len(msgs) == 0 {
			w.Line("  (no messages)")
		}
		for i := lo; i < hi; i++ {
			m := msgs[i]
			w.Color(screen.Yellow)
			w.Printf("  %d) ", i+1)
			w.Color(screen.White)
			w.SafePrint(firstLine(m.Subject))
			w.Reset()
			w.Printf("  — %s, %s\r\n", handles.handle(m.AuthorID), m.PostedAt.Format("2006-01-02"))
		}
		pageFooter(w, page, pages)
		w.Color(screen.Green)
		w.Print("\r\n[#] read  [P]ost  [S]earch  [U] by user  [D] sort  [C]lear  [Q]uit: ")
		w.Reset()
		in, err := s.ReadLine()
		if err != nil {
			return err
		}
		in = strings.TrimSpace(in)
		switch {
		case in == "" || strings.EqualFold(in, "q"):
			return nil
		case in == ">":
			page++
		case in == "<":
			page--
		case strings.EqualFold(in, "p"):
			listed, listedHeader, page = nil, "", 0
			if err := compose(s, st, u, area.ID, nil); err != nil {
				return err
			}
		case strings.EqualFold(in, "d"):
			newestFirst = !newestFirst
			listed, listedHeader, page = nil, "", 0
		case strings.EqualFold(in, "c"):
			listed, listedHeader, page = nil, "", 0
		case strings.EqualFold(in, "s"):
			w.Color(screen.Green)
			w.Print("\r\nSearch text: ")
			w.Reset()
			q, _ := s.ReadLine()
			q = strings.TrimSpace(q)
			if q != "" {
				res, serr := st.Messages().Search(area.ID, q)
				if serr != nil {
					return serr
				}
				listed, listedHeader, page = res, fmt.Sprintf("search %q — %d hit(s)", q, len(res)), 0
			}
		case strings.EqualFold(in, "u"):
			w.Color(screen.Green)
			w.Print("\r\nFilter by handle: ")
			w.Reset()
			h, _ := s.ReadLine()
			h = strings.TrimSpace(h)
			if target, terr := st.Users().ByHandle(h); terr == nil {
				res, ferr := st.Messages().ByAuthor(area.ID, target.ID)
				if ferr != nil {
					return ferr
				}
				listed, listedHeader, page = res, fmt.Sprintf("by %s — %d message(s)", target.Handle, len(res)), 0
			} else {
				w.ColorLine(screen.Red, "no such user")
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
	replies = hideBlocked(st, u.ID, replies)
	for _, rep := range replies {
		w.Color(screen.Blue)
		w.Printf("\r\n  ┌ reply from %s, %s\r\n", handles.handle(rep.AuthorID), rep.PostedAt.Format("2006-01-02"))
		w.Reset()
		w.Print("  ")
		w.SafePrint(strings.ReplaceAll(rep.Body, "\n", "\n  "))
		w.Print("\r\n")
	}

	w.Color(screen.Green)
	w.Print("\r\n[R]eply  [B]lock author  re[P]ort  [Q]uit: ")
	w.Reset()
	key, err := s.ReadKey()
	if err != nil {
		return err
	}
	switch toLower(key) {
	case 'r':
		return compose(s, st, u, m.AreaID, &m.ID)
	case 'b':
		if m.AuthorID == u.ID {
			break
		}
		if err := st.Blocks().Block(u.ID, m.AuthorID); err != nil {
			w.ColorLine(screen.Red, "could not block")
		} else {
			s.Activity("block-user", handles.handle(m.AuthorID))
			w.ColorLine(screen.Cyan, "\r\nBlocked. Their posts are now hidden from you.")
			_, _ = s.ReadKey()
		}
	case 'p':
		reportUser(s, st, u, m.AuthorID, fmt.Sprintf("board post #%d", m.ID), handles)
	}
	return nil
}

// hideBlocked drops posts authored by users the viewer has blocked.
func hideBlocked(st *store.Store, viewerID int64, msgs []*store.Message) []*store.Message {
	blocked, err := st.Blocks().BlockedSet(viewerID)
	if err != nil || len(blocked) == 0 {
		return msgs
	}
	kept := msgs[:0]
	for _, m := range msgs {
		if !blocked[m.AuthorID] {
			kept = append(kept, m)
		}
	}
	return kept
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
