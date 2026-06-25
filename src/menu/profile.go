package menu

import (
	"errors"
	"strconv"
	"strings"

	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// RunProfile lets a member manage their own SSH keys (list / add / revoke) and
// change their password.
func RunProfile(s *session.Session, st *store.Store, u *store.User) error {
	keys := st.Keys()
	for {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		w.Clear()
		// Stats — re-read from the store so SysOp-awarded points show live.
		stats := u
		if fresh, err := st.Users().ByID(u.ID); err == nil {
			stats = fresh
		}
		w.ColorLine(screen.Cyan, "Your Stats")
		w.ColorLine(screen.Cyan, "----------")
		w.Printf("  Handle: %s   Access level: %d\r\n", stats.Handle, stats.AccessLevel)
		w.Printf("  Points: %d   Member since: %s\r\n\r\n", stats.Points, stats.CreatedAt.Format("2006-01-02"))
		w.ColorLine(screen.Cyan, "Your SSH Keys")
		w.ColorLine(screen.Cyan, "-------------")
		active, err := keys.Active(u.ID)
		if err != nil {
			return err
		}
		if len(active) == 0 {
			w.Line("  (none — add one before you lose access!)")
		}
		for i, k := range active {
			w.Printf("  %d) %s  %s\r\n", i+1, k.Fingerprint, k.Comment)
		}
		w.Color(screen.Green)
		w.Print("\r\n[A]dd key  [R]evoke key  [P]assword  [B]locked users  [Q]uit: ")
		w.Reset()

		key, err := s.ReadKey()
		if err != nil {
			return err
		}
		switch toLower(key) {
		case 'p':
			if err := changePassword(s, st, u, w); err != nil {
				return err
			}
		case 'b':
			if err := manageBlocks(s, st, u); err != nil {
				return err
			}
		case 'a':
			w.Print("\r\nPaste public key: ")
			line, err := s.ReadLine()
			if err != nil {
				return err
			}
			if _, err := keys.Add(u.ID, strings.TrimSpace(line)); err != nil {
				if errors.Is(err, store.ErrKeyTaken) {
					w.ColorLine(screen.Red, "  [x] that key is already registered to an account")
				} else {
					w.ColorLine(screen.Red, "  [x] invalid SSH public key")
				}
			} else {
				w.ColorLine(screen.Cyan, "  [ok] key added")
				s.Activity("key-add", "")
			}
		case 'r':
			if len(active) == 0 {
				continue
			}
			w.Print("\r\nNumber to revoke (or blank to cancel): ")
			in, err := s.ReadLine()
			if err != nil {
				return err
			}
			n, perr := strconv.Atoi(strings.TrimSpace(in))
			if perr != nil || n < 1 || n > len(active) {
				w.ColorLine(screen.Red, "  [x] no such key")
				continue
			}
			if err := keys.Revoke(active[n-1].ID); err != nil {
				w.ColorLine(screen.Red, "  [x] could not revoke")
			} else {
				w.ColorLine(screen.Cyan, "  [ok] key revoked")
				s.Activity("key-revoke", active[n-1].Fingerprint)
			}
		case 'q':
			return nil
		}
	}
}

// manageBlocks lets a member view their block list, block a user by handle, and
// unblock. Blocking can also be done contextually from mail/board read views.
func manageBlocks(s *session.Session, st *store.Store, u *store.User) error {
	for {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		w.Clear()
		w.ColorLine(screen.Cyan, "Blocked Users")
		w.ColorLine(screen.Cyan, "-------------")
		ids, err := st.Blocks().List(u.ID)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			w.Line("  (you haven't blocked anyone)")
		}
		handles := make([]string, len(ids))
		for i, id := range ids {
			h := "unknown"
			if bu, err := st.Users().ByID(id); err == nil {
				h = bu.Handle
			}
			handles[i] = h
			w.Printf("  %d) %s\r\n", i+1, h)
		}
		w.Color(screen.Green)
		w.Print("\r\n[A]dd by handle  [U]nblock #  [Q]uit: ")
		w.Reset()
		key, err := s.ReadKey()
		if err != nil {
			return err
		}
		switch toLower(key) {
		case 'a':
			w.Print("\r\nHandle to block: ")
			in, err := s.ReadLine()
			if err != nil {
				return err
			}
			target, terr := st.Users().ByHandle(strings.TrimSpace(in))
			if terr != nil {
				w.ColorLine(screen.Red, "  [x] no such user")
			} else if target.ID == u.ID {
				w.ColorLine(screen.Red, "  [x] you can't block yourself")
			} else if err := st.Blocks().Block(u.ID, target.ID); err != nil {
				w.ColorLine(screen.Red, "  [x] could not block")
			} else {
				s.Activity("block-user", target.Handle)
				w.ColorLine(screen.Cyan, "  [ok] blocked")
			}
		case 'u':
			if len(ids) == 0 {
				continue
			}
			w.Print("\r\nNumber to unblock (or blank to cancel): ")
			in, err := s.ReadLine()
			if err != nil {
				return err
			}
			n, perr := strconv.Atoi(strings.TrimSpace(in))
			if perr != nil || n < 1 || n > len(ids) {
				continue
			}
			if err := st.Blocks().Unblock(u.ID, ids[n-1]); err != nil {
				w.ColorLine(screen.Red, "  [x] could not unblock")
			} else {
				s.Activity("unblock-user", handles[n-1])
				w.ColorLine(screen.Cyan, "  [ok] unblocked")
			}
		case 'q':
			return nil
		}
	}
}

func changePassword(s *session.Session, st *store.Store, u *store.User, w *screen.Writer) error {
	w.Color(screen.Green)
	w.Print("\r\nCurrent password: ")
	w.Reset()
	cur, err := s.ReadPassword()
	if err != nil {
		return err
	}
	if ok, _ := store.VerifyPassword(u.PasswordHash, cur); !ok {
		w.ColorLine(screen.Red, "wrong password")
		_, _ = s.ReadKey()
		return nil
	}
	w.Color(screen.Green)
	w.Print("New password (min 8): ")
	w.Reset()
	p1, err := s.ReadPassword()
	if err != nil {
		return err
	}
	w.Color(screen.Green)
	w.Print("Confirm: ")
	w.Reset()
	p2, err := s.ReadPassword()
	if err != nil {
		return err
	}
	if p1 != p2 || len(p1) < minPasswordLen {
		w.ColorLine(screen.Red, "passwords didn't match or were too short")
		_, _ = s.ReadKey()
		return nil
	}
	hash, err := store.HashPassword(p1)
	if err != nil {
		return err
	}
	if err := st.Users().SetPassword(u.ID, hash); err != nil {
		return err
	}
	u.PasswordHash = hash
	s.Activity("password-change", "")
	w.ColorLine(screen.Cyan, "Password changed.")
	_, _ = s.ReadKey()
	return nil
}
