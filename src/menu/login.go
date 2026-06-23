package menu

import (
	"strings"
	"time"

	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// minPasswordLen is the floor for a member password.
const minPasswordLen = 8

// genericLoginFail avoids leaking whether a handle exists (SEC-4).
const genericLoginFail = "login failed"

// RunLogin authenticates the second factor over the (already key-authenticated)
// SSH channel. The SSH PublicKeyCallback has proven the caller holds a
// registered key for this handle; here we add the password. On a first login
// (no password yet) we run onboarding: redeem the one-time token, then set a
// password. Returns the logged-in user and true on success.
func RunLogin(s *session.Session, st *store.Store) (*store.User, bool) {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)

	u, err := st.Users().ByHandle(s.Username())
	if err != nil || u.Status != store.StatusApproved {
		// Defence in depth — the pubkey gate should already have stopped this.
		w.ColorLine(screen.Red, genericLoginFail)
		return nil, false
	}

	if u.PasswordHash == "" {
		return onboard(s, w, st, u)
	}
	return verify(s, w, st, u)
}

func onboard(s *session.Session, w *screen.Writer, st *store.Store, u *store.User) (*store.User, bool) {
	w.Clear()
	w.ColorLine(screen.Cyan, "Welcome, "+u.Handle+" — first login.")
	w.Line("Enter the one-time token your SysOp sent you.")
	w.Color(screen.Green)
	w.Print("Token: ")
	w.Reset()
	tok, err := s.ReadLine()
	if err != nil {
		return nil, false
	}
	if err := st.Tokens().Redeem(u.ID, strings.TrimSpace(tok)); err != nil {
		w.ColorLine(screen.Red, "invalid or expired token")
		s.Activity("onboard-failed", "bad token")
		return nil, false
	}

	w.Print("\r\n")
	w.Line("Set your password (min 8 chars). It is never sent in the clear again.")
	w.Color(screen.Green)
	w.Print("New password: ")
	w.Reset()
	p1, err := s.ReadPassword()
	if err != nil {
		return nil, false
	}
	w.Color(screen.Green)
	w.Print("Confirm password: ")
	w.Reset()
	p2, err := s.ReadPassword()
	if err != nil {
		return nil, false
	}
	if p1 != p2 || len(p1) < minPasswordLen {
		w.ColorLine(screen.Red, "passwords did not match or were too short")
		return nil, false
	}
	hash, err := store.HashPassword(p1)
	if err != nil {
		w.ColorLine(screen.Red, "could not set password")
		return nil, false
	}
	if err := st.Users().SetPassword(u.ID, hash); err != nil {
		w.ColorLine(screen.Red, "could not set password")
		return nil, false
	}
	st.Users().TouchLogin(u.ID, time.Now())
	s.Activity("onboarded", "")
	w.ColorLine(screen.Cyan, "Password set. Welcome aboard!")
	u.PasswordHash = hash
	return u, true
}

func verify(s *session.Session, w *screen.Writer, st *store.Store, u *store.User) (*store.User, bool) {
	for attempt := 0; attempt < 3; attempt++ {
		w.Color(screen.Green)
		w.Print("Password: ")
		w.Reset()
		pw, err := s.ReadPassword()
		if err != nil {
			return nil, false
		}
		ok, _ := store.VerifyPassword(u.PasswordHash, pw)
		if ok {
			st.Users().TouchLogin(u.ID, time.Now())
			s.Activity("login", "ok")
			return u, true
		}
		s.Activity("login-failed", "")
		w.ColorLine(screen.Red, genericLoginFail)
		time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond) // backoff (SEC-4)
	}
	return nil, false
}
