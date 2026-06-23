package menu

import (
	"strconv"
	"strings"

	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// RunProfile lets a member manage their own SSH keys (list / add / revoke).
func RunProfile(s *session.Session, u *store.User, keys *store.Keys) error {
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
		w.Print("\r\n[A]dd  [R]evoke  [Q]uit: ")
		w.Reset()

		key, err := s.ReadKey()
		if err != nil {
			return err
		}
		switch toLower(key) {
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
