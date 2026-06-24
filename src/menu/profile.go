package menu

import (
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
		w.Print("\r\n[A]dd key  [R]evoke key  [P]assword  [Q]uit: ")
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
		case 'a':
			w.Print("\r\nPaste public key: ")
			line, err := s.ReadLine()
			if err != nil {
				return err
			}
			if _, err := keys.Add(u.ID, strings.TrimSpace(line)); err != nil {
				w.ColorLine(screen.Red, "  [x] invalid SSH public key")
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
