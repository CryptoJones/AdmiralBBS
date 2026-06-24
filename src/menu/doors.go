package menu

import (
	"strconv"
	"strings"
	"time"

	"admiralbbs/src/doors"
	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// RunDoors lists the door games a member may play and launches the chosen one
// as a sandboxed subprocess wired to the session.
func RunDoors(s *session.Session, st *store.Store, u *store.User) error {
	for {
		cap := s.Cap()
		w := screen.New(s, cap.ANSI, cap.Cols)
		w.Clear()
		w.ColorLine(screen.Cyan, "Door Games")
		w.ColorLine(screen.Cyan, "----------")
		list, err := st.Doors().Visible(u.AccessLevel)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			w.Line("  (no door games installed yet)")
		}
		for i, d := range list {
			w.Color(screen.Yellow)
			w.Printf("  %d) ", i+1)
			w.Color(screen.White)
			w.SafePrint(d.Name)
			w.Reset()
			w.Print("\r\n")
		}
		w.Color(screen.Green)
		w.Print("\r\nPlay # (or [Q]uit): ")
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
		if perr != nil || n < 1 || n > len(list) {
			continue
		}
		if err := playDoor(s, u, list[n-1]); err != nil {
			return err
		}
	}
}

func playDoor(s *session.Session, u *store.User, d *store.Door) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.Color(screen.Magenta)
	w.Print("Launching ")
	w.SafePrint(d.Name)
	w.Print("...\r\n\r\n")
	w.Reset()
	s.Activity("door-launch", d.Name)

	drop := doors.DropInfo{
		BBSName:     "AdmiralBBS",
		Handle:      u.Handle,
		AccessLevel: u.AccessLevel,
		MinutesLeft: 30,
		Node:        1,
		ANSI:        cap.ANSI,
	}
	err := doors.Launch(s.Raw(), d.Command, nil, drop, doors.Opts{Timeout: 15 * time.Minute, Term: termOf(cap.ANSI)})

	w.Color(screen.Cyan)
	w.Print("\r\n\r\n")
	w.SafePrint(d.Name)
	w.Print(" exited. Press any key...")
	w.Reset()
	if err != nil {
		s.Activity("door-exit", d.Name+": "+err.Error())
	}
	_, kerr := s.ReadKey()
	return kerr
}

func termOf(ansi bool) string {
	if ansi {
		return "ansi"
	}
	return "dumb"
}
