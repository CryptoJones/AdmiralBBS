package tests

import (
	"testing"

	"admiralbbs/src/session"
	"admiralbbs/src/transport"
)

func TestDetectCapability(t *testing.T) {
	cases := []struct {
		term     string
		ws       transport.WindowSize
		wantANSI bool
		wantCols int
	}{
		{"ansi", transport.WindowSize{Cols: 80, Rows: 25}, true, 80},
		{"xterm-256color", transport.WindowSize{Cols: 132, Rows: 50}, true, 132},
		{"syncterm", transport.WindowSize{}, true, 80},     // unknown size → 80
		{"dumb", transport.WindowSize{Cols: 80}, false, 80}, // forced B&W
		{"vt52", transport.WindowSize{}, false, 80},
		{"", transport.WindowSize{}, true, 80}, // no type → assume ANSI per spec
		{"weirdterm9000", transport.WindowSize{}, false, 80}, // unknown present → B&W
	}
	for _, c := range cases {
		got := session.DetectCapability(c.term, c.ws)
		if got.ANSI != c.wantANSI {
			t.Errorf("term %q: ANSI=%v, want %v", c.term, got.ANSI, c.wantANSI)
		}
		if got.Cols != c.wantCols {
			t.Errorf("term %q: Cols=%d, want %d", c.term, got.Cols, c.wantCols)
		}
	}
}
