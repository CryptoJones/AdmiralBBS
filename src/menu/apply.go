package menu

import (
	"strings"

	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// RunApply is the Telnet-only membership application. It collects a handle,
// one or more SSH public keys, an optional contact, and a note, then creates a
// PENDING user (no password — set later over SSH) and a membership application.
//
// No secret is ever collected here: Telnet is plaintext, so only public data
// (handle, public keys, contact) is gathered. The password is set on first SSH
// login behind the one-time approval token (DECISIONS: Telnet = apply-only).
func RunApply(s *session.Session, st *store.Store) error {
	users, memberships, keys := st.Users(), st.Memberships(), st.Keys()
	title := st.Settings().BBSName() + " :: Membership Application"
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Cyan, title)
	w.ColorLine(screen.Cyan, strings.Repeat("-", len(title)))
	w.Line("Telnet is for applying only. Approved members connect over SSH.")
	w.Print("\r\n")

	w.Color(screen.Green)
	w.Print("Desired handle: ")
	w.Reset()
	handle, err := s.ReadLine()
	if err != nil {
		return err
	}
	handle = strings.TrimSpace(handle)
	if handle == "" {
		w.Line("No handle entered. Goodbye.")
		return nil
	}
	if _, err := users.ByHandle(handle); err == nil {
		w.ColorLine(screen.Red, "That handle is already taken. Goodbye.")
		return nil
	}

	w.Print("\r\n")
	w.Line("Paste your SSH PUBLIC key(s), one per line. Blank line when done.")
	w.Line("(Public keys are safe to send in the clear; your password is set later over SSH.)")
	var keyLines []string
	for {
		w.Color(screen.Green)
		w.Print("key> ")
		w.Reset()
		line, err := s.ReadLine()
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if verr := store.ValidatePublicKey(line); verr != nil {
			w.ColorLine(screen.Red, "  [x] not a valid SSH public key - skipped")
			continue
		}
		keyLines = append(keyLines, line)
		w.ColorLine(screen.Cyan, "  [ok] key accepted")
	}
	if len(keyLines) == 0 {
		w.ColorLine(screen.Red, "At least one SSH public key is required to join. Goodbye.")
		return nil
	}

	w.Print("\r\n")
	w.Line("How can the SysOp reach you out-of-band to deliver your approval")
	w.Color(screen.Green)
	w.Print("token? (optional, e.g. PGP-encrypted email): ")
	w.Reset()
	contact, err := s.ReadLine()
	if err != nil {
		return err
	}

	w.Color(screen.Green)
	w.Print("Why do you want to join? ")
	w.Reset()
	note, err := s.ReadLine()
	if err != nil {
		return err
	}

	// Persist: pending user (empty password), keys, then the application.
	u, err := users.Create(handle, "", "", "")
	if err != nil {
		w.ColorLine(screen.Red, "Could not submit application: "+err.Error())
		return nil
	}
	for _, k := range keyLines {
		if _, err := keys.Add(u.ID, k); err != nil {
			w.ColorLine(screen.Red, "  [x] a key failed to save: "+err.Error())
		}
	}
	fullNote := strings.TrimSpace(note)
	if c := strings.TrimSpace(contact); c != "" {
		fullNote = "contact: " + c + "\n" + fullNote
	}
	if _, err := memberships.Apply(u.ID, fullNote); err != nil {
		w.ColorLine(screen.Red, "Could not record application: "+err.Error())
		return nil
	}

	s.Activity("membership-application", "handle="+handle)
	w.Print("\r\n")
	w.ColorLine(screen.Cyan, "Application submitted! The SysOp will review it.")
	w.Line("Once approved you'll receive a one-time token out-of-band, then")
	w.Line("connect via SSH to set your password and log in. 73!")
	return nil
}
