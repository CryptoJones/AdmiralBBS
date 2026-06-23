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
	Title   string
	ArtPath string // optional CP437 .ans banner
	Items   []Item
}

// ErrDisconnected is returned when the caller drops mid-menu.
var ErrDisconnected = errors.New("caller disconnected")

// Run renders the menu and dispatches keypresses until an action returns
// Logoff or the caller disconnects.
func (m *Menu) Run(s *session.Session) error {
	for {
		m.render(s)
		key, err := s.ReadKey()
		if err != nil {
			return ErrDisconnected
		}
		item := m.find(key)
		if item == nil {
			continue // unknown key: redraw
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
	cap := s.Cap()
	w := screen.New(s, cap.ANSI, cap.Cols)
	w.Clear()

	if m.ArtPath != "" {
		if art, err := screen.LoadArt(m.ArtPath); err == nil && len(art) > 0 {
			screen.RenderArt(w, art)
			w.Print("\r\n")
		}
	}

	w.ColorLine(screen.Cyan, m.Title)
	w.ColorLine(screen.Cyan, strings.Repeat("-", len(m.Title)))
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
