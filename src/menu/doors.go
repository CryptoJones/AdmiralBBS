package menu

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"admiralbbs/src/doors"
	"admiralbbs/src/screen"
	"admiralbbs/src/session"
	"admiralbbs/src/store"
)

// RunDoors lists the door games a member may play and launches the chosen one
// as a sandboxed subprocess wired to the session. base carries the deploy-time
// isolation options (uid/chroot/namespaces); per-launch Timeout/Term are added.
func RunDoors(s *session.Session, st *store.Store, u *store.User, base doors.Opts, node int, doorsData string) error {
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
		if err := playDoor(s, u, list[n-1], base, node, doorsData); err != nil {
			return err
		}
	}
}

func playDoor(s *session.Session, u *store.User, d *store.Door, base doors.Opts, node int, doorsData string) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	// printLaunch renders the "Launching <name> [vX.Y.Z] (node N)..." banner.
	// The version is shown only when the door advertised one via the resident
	// handshake (ABBS Door Spec §2.2); subprocess doors have no version channel.
	printLaunch := func(ver string) {
		w.Color(screen.Magenta)
		w.Print("Launching ")
		w.SafePrint(d.Name)
		if ver != "" {
			w.Printf(" v%s", ver)
		}
		w.Printf(" (node %d)...\r\n\r\n", node)
		w.Reset()
		s.Activity("door-launch", d.Name)
	}

	var err error
	if d.Kind == store.KindResident {
		// Persistent multiplayer server (MajorMUD-style): relay to the one
		// running game so all callers share the world. Dial first so we can read
		// the door's optional version handshake and surface it on the launch line.
		net := d.Network
		if net == "" {
			net = "tcp"
		}
		rc, derr := doors.DialResident(net, d.Address, 10*time.Second, 1500*time.Millisecond)
		if derr != nil {
			printLaunch("")
			err = derr
		} else {
			printLaunch(rc.Version)
			rc.SendHandle(u.Handle) // door defaults its name prompt to the BBS handle (if it asked)
			err = rc.Relay(s.Raw())
			rc.Close()
		}
	} else {
		printLaunch("")
		slug := slugify(d.Name)
		opts := base
		opts.Timeout = 15 * time.Minute
		opts.Term = termOf(cap.ANSI)
		opts.WorkDir = filepath.Join(doorsData, slug, fmt.Sprintf("node%d", node)) // per-node, persistent
		opts.ShareDir = filepath.Join(doorsData, slug, "shared")                   // shared multiplayer state
		drop := doors.DropInfo{
			BBSName:     "AdmiralBBS",
			Handle:      u.Handle,
			AccessLevel: u.AccessLevel,
			MinutesLeft: 30,
			Node:        node,
			ANSI:        cap.ANSI,
		}
		err = doors.Launch(s.Raw(), d.Command, nil, drop, opts)
	}

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

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "door"
	}
	return out
}

func termOf(ansi bool) string {
	if ansi {
		return "ansi"
	}
	return "dumb"
}
