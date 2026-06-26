package tests

import (
	"bytes"
	"strings"
	"testing"

	"admiralbbs/src/screen"
)

// cbWriter is an io.Writer that also reports Colorblind() — mimics the session.
type cbWriter struct {
	bytes.Buffer
	cb bool
}

func (c *cbWriter) Colorblind() bool { return c.cb }

// TestColorblindRemap checks the screen Writer remaps green→blue when the
// underlying writer reports colorblind, and leaves it alone otherwise (#9).
func TestColorblindRemap(t *testing.T) {
	// Default: green stays green (SGR 32).
	var def cbWriter
	screen.New(&def, true, 80).ColorLine(screen.Green, "ok")
	if !strings.Contains(def.String(), "\x1b[32m") {
		t.Errorf("default should emit green (32); got %q", def.String())
	}

	// Colorblind: green → blue (34), not green (32).
	cb := cbWriter{cb: true}
	screen.New(&cb, true, 80).ColorLine(screen.Green, "ok")
	got := cb.String()
	if strings.Contains(got, "\x1b[32m") {
		t.Errorf("colorblind should not emit raw green (32); got %q", got)
	}
	if !strings.Contains(got, "\x1b[34m") {
		t.Errorf("colorblind should remap success to blue (34); got %q", got)
	}
}

// TestColorblindPersist checks the user colorblind setter round-trips (#9).
func TestColorblindPersist(t *testing.T) {
	s, _ := openTestStore(t)
	u, _ := s.Users().Create("dave", "x", "", "")
	if u.Colorblind {
		t.Fatal("new user should default to colorblind off")
	}
	if err := s.Users().SetColorblind(u.ID, true); err != nil {
		t.Fatalf("SetColorblind: %v", err)
	}
	again, _ := s.Users().ByID(u.ID)
	if !again.Colorblind {
		t.Error("colorblind preference should persist")
	}
}
