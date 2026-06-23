package tests

import (
	"bytes"
	"strings"
	"testing"

	"admiralbbs/src/screen"
)

func TestWriterANSIEmitsEscapes(t *testing.T) {
	var buf bytes.Buffer
	w := screen.New(&buf, true, 80)
	w.Clear()
	w.ColorLine(screen.Red, "hi")
	if !bytes.Contains(buf.Bytes(), []byte{0x1b}) {
		t.Fatal("ANSI writer emitted no escape codes")
	}
}

func TestWriterBWNeverEmitsEscapes(t *testing.T) {
	var buf bytes.Buffer
	w := screen.New(&buf, false, 80)
	w.Clear()
	w.Color(screen.Red)
	w.Bold()
	w.ColorLine(screen.Green, "plain")
	w.Reset()
	if bytes.Contains(buf.Bytes(), []byte{0x1b}) {
		t.Fatalf("B&W writer leaked an escape byte: %q", buf.Bytes())
	}
	if !strings.Contains(buf.String(), "plain") {
		t.Fatal("B&W writer dropped the text content")
	}
}

func TestRenderArtDegradesToBW(t *testing.T) {
	// ANSI colour codes + CP437 box-drawing should become plain ASCII.
	art := append([]byte("\x1b[1;36m"), 0xC9, 0xCD, 0xCD, 0xBB)
	art = append(art, []byte("\x1b[0mTITLE")...)

	var buf bytes.Buffer
	w := screen.New(&buf, false, 80)
	screen.RenderArt(w, art)

	out := buf.Bytes()
	if bytes.Contains(out, []byte{0x1b}) {
		t.Fatalf("degraded art leaked escape byte: %q", out)
	}
	for _, b := range out {
		if b >= 0x80 {
			t.Fatalf("degraded art leaked high CP437 byte %d", b)
		}
	}
	if !bytes.Contains(out, []byte("TITLE")) {
		t.Fatal("degraded art dropped text")
	}
	if !bytes.Contains(out, []byte("+--+")) {
		t.Fatalf("CP437 box not folded to ASCII: %q", out)
	}
}
