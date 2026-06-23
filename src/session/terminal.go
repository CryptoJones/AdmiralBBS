package session

import (
	"strings"

	"admiralbbs/src/transport"
)

// Capability describes what the caller's terminal can do. The BBS renders in
// ANSI colour when ANSI is true and degrades to plain black-and-white text
// otherwise (DECISIONS.md: ANSI when available, B&W for older terminals).
type Capability struct {
	ANSI  bool // colour + cursor control supported
	CP437 bool // IBM code page 437 box-drawing assumed
	Cols  int  // columns; 80 when unknown
	Rows  int  // rows; 0 when unknown — never assumed for layout
}

// ansiTermHints are terminal-type substrings that imply ANSI capability.
var ansiTermHints = []string{
	"ansi", "xterm", "vt100", "vt102", "screen", "tmux",
	"syncterm", "netrunner", "cygwin", "linux", "putty", "rxvt",
}

// plainTermHints force black-and-white (older / non-ANSI terminals).
var plainTermHints = []string{"dumb", "vt52", "mono", "net", "unknown"}

// DetectCapability derives the rendering capability from the negotiated
// terminal type and window size.
func DetectCapability(termType string, ws transport.WindowSize) Capability {
	cols := ws.Cols
	if cols <= 0 {
		cols = 80 // ANSI-BBS: 80 columns is the safe assumption
	}

	cap := Capability{Cols: cols, Rows: ws.Rows}

	t := strings.ToLower(strings.TrimSpace(termType))
	switch {
	case t == "":
		// No type negotiated. Most BBS callers are ANSI-capable; assume ANSI
		// + CP437 per the spec, but the writer still degrades safely.
		cap.ANSI = true
		cap.CP437 = true
	case containsAny(t, plainTermHints):
		cap.ANSI = false
		cap.CP437 = false
	case containsAny(t, ansiTermHints):
		cap.ANSI = true
		cap.CP437 = true
	default:
		// Unknown but present type: be conservative, render B&W.
		cap.ANSI = false
		cap.CP437 = false
	}
	return cap
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
