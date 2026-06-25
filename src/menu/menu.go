// Package menu is the data-driven menu engine: a menu is a list of items, a
// keypress is dispatched to the matching item's action. It renders through the
// screen package so it inherits ANSI/B&W degradation automatically.
package menu

import (
	"errors"
	"strings"

	"admiralbbs/src/screen"
	"admiralbbs/src/session"
)

// Outcome tells the engine what to do after an item's action returns.
type Outcome int

const (
	Continue Outcome = iota // redraw and stay in this menu
	Logoff                  // disconnect the caller
)

// Action runs when its item is chosen.
type Action func(*session.Session) (Outcome, error)

// Item is one selectable menu entry.
type Item struct {
	Key    byte // hotkey (matched case-insensitively)
	Label  string
	Action Action
}

// Menu is a titled (optionally art-backed) list of items.
type Menu struct {
	Title    string
	Subtitle string   // optional line under the title (e.g. the BBS tagline)
	Banner   []string // optional generated banner lines (shown when ArtPath is unset)
	ArtPath  string   // optional CP437 .ans banner (a SysOp's custom art; overrides Banner)
	Items    []Item
	// Refresh, if set, runs before each render to recompute dynamic fields
	// (title/banner/items) — so live config changes show without a relog.
	Refresh func()
	// OnQuit, if set, runs when the caller presses a reserved quit key ([Q] or
	// [X]) that no item claims — so X and Q always back out of a menu. On the
	// top-level menu this is the logoff action.
	OnQuit Action
}

// BBSBanner builds a simple, brand-neutral ASCII banner from the configured BBS
// name + tagline — so the login screen reflects *this* BBS, not a hardcoded one.
// A SysOp who wants fancier art supplies their own .ans via -art (which wins).
func BBSBanner(name, tagline string) []string {
	title := spaceOut(strings.ToUpper(strings.TrimSpace(name)))
	if title == "" {
		title = "B B S"
	}
	width := len(title)
	if len(tagline) > width {
		width = len(tagline)
	}
	width += 6
	bar := strings.Repeat("=", width)
	lines := []string{bar, center(title, width)}
	if t := strings.TrimSpace(tagline); t != "" {
		lines = append(lines, center(t, width))
	}
	return append(lines, bar)
}

// spaceOut inserts a space between characters ("BBS" -> "B B S") for a retro
// banner feel, unless the result would be unreasonably wide.
func spaceOut(s string) string {
	if len(s)*2 > 70 {
		return s
	}
	r := []rune(s)
	out := make([]rune, 0, len(r)*2)
	for i, c := range r {
		if i > 0 {
			out = append(out, ' ')
		}
		out = append(out, c)
	}
	return string(out)
}

func center(s string, width int) string {
	if len(s) >= width {
		return s
	}
	left := (width - len(s)) / 2
	return strings.Repeat(" ", left) + s
}

// ShowMOTD displays the message of the day and waits for the caller to press
// ANY key to confirm they've read it (they can re-read it from the main menu if
// they blew past it).
func ShowMOTD(s *session.Session, motd string) error {
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()
	w.ColorLine(screen.Cyan, "* Message of the Day *")
	w.ColorLine(screen.Cyan, "----------------------")
	for _, line := range strings.Split(motd, "\n") {
		w.SafePrint(strings.TrimRight(line, "\r"))
		w.Print("\r\n")
	}
	w.Color(screen.Green)
	w.Print("\r\nPress any key to continue...")
	w.Reset()
	// Any key dismisses the MOTD (not just SPACE).
	if _, err := s.ReadKey(); err != nil {
		return err
	}
	return nil
}

// ErrDisconnected is returned when the caller drops mid-menu.
var ErrDisconnected = errors.New("caller disconnected")

// Run renders the menu and dispatches keypresses until an action returns
// Logoff or the caller disconnects.
func (m *Menu) Run(s *session.Session) error {
	for {
		m.render(s)
		// Read a full line (CR-terminated) so the main menu matches the rest of
		// the BBS — you type the command letter and press Enter.
		line, err := s.ReadLine()
		if err != nil {
			return ErrDisconnected
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue // bare Enter: redraw
		}
		item := m.find(line[0])
		if item == nil {
			// X and Q are reserved quit keys: if nothing claims them, back out.
			if k := toLower(line[0]); (k == 'q' || k == 'x') && m.OnQuit != nil {
				out, err := m.OnQuit(s)
				if err != nil {
					return err
				}
				if out == Logoff {
					return nil
				}
			}
			continue // unknown command: redraw
		}
		out, err := item.Action(s)
		if err != nil {
			return err
		}
		if out == Logoff {
			return nil
		}
	}
}

func (m *Menu) find(key byte) *Item {
	k := toLower(key)
	for i := range m.Items {
		if toLower(m.Items[i].Key) == k {
			return &m.Items[i]
		}
	}
	return nil
}

func (m *Menu) render(s *session.Session) {
	if m.Refresh != nil {
		m.Refresh() // pull live config (branding, items) before drawing
	}
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()

	shownArt := false
	if m.ArtPath != "" {
		if art, err := screen.LoadArt(m.ArtPath); err == nil && len(art) > 0 {
			screen.RenderArt(w, art)
			w.Print("\r\n")
			shownArt = true
		}
	}
	if !shownArt && len(m.Banner) > 0 {
		for _, ln := range m.Banner {
			w.ColorLine(screen.Cyan, ln)
		}
		w.Print("\r\n")
	}

	w.ColorLine(screen.Cyan, m.Title)
	w.ColorLine(screen.Cyan, strings.Repeat("-", len(m.Title)))
	if m.Subtitle != "" {
		w.ColorLine(screen.Blue, m.Subtitle)
		w.Print("\r\n")
	}
	for _, it := range m.Items {
		w.Color(screen.Yellow)
		w.Printf("  [%c] ", upper(it.Key))
		w.Color(screen.White)
		w.Print(it.Label)
		w.Reset()
		w.Print("\r\n")
	}
	w.Color(screen.Green)
	w.Print("\r\nCommand: ")
	w.Reset()
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func upper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - ('a' - 'A')
	}
	return b
}
