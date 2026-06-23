package tests

import (
	"bytes"
	"testing"

	"admiralbbs/src/session"
)

// Hardening: the sanitiser must strip control bytes and escape sequences
// (the "packet injection" class) while preserving printable + CP437 bytes.

func TestSanitizeDropsControlExceptWhitelist(t *testing.T) {
	in := []byte{0x00, 0x01, 'h', 0x07, 'i', '\t', '\r', '\n', 0x7F}
	got := session.SanitizeInput(in)
	want := []byte{'h', 'i', '\t', '\r', '\n'}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSanitizeStripsCSISequences(t *testing.T) {
	// ESC[31m RED ESC[0m  → only "RED" survives
	in := append([]byte("\x1b[31mRED\x1b[0m"), '!')
	got := session.SanitizeInput(in)
	if string(got) != "RED!" {
		t.Fatalf("got %q, want %q", got, "RED!")
	}
}

func TestSanitizeStripsLoneEscapeFunction(t *testing.T) {
	// ESC + 'c' (RIS, terminal reset) must not survive.
	in := []byte("a\x1bcb")
	got := session.SanitizeInput(in)
	if string(got) != "ab" {
		t.Fatalf("got %q, want %q", got, "ab")
	}
}

func TestSanitizePreservesHighCP437(t *testing.T) {
	in := []byte{0xC9, 0xCD, 0xBB} // CP437 box-drawing corners
	got := session.SanitizeInput(in)
	if !bytes.Equal(got, in) {
		t.Fatalf("high bytes were altered: got %v", got)
	}
}

func TestSanitizeNeverEmitsControlOrEscape(t *testing.T) {
	in := []byte("\x1b[2J\x00\x01hello\x07\x1b[Hworld\x7f")
	got := session.SanitizeInput(in)
	for _, b := range got {
		if b == 0x1B {
			t.Fatalf("escape byte survived: %q", got)
		}
		if (b < 0x20 || b == 0x7F) && b != '\t' && b != '\r' && b != '\n' {
			t.Fatalf("disallowed control byte %d survived: %q", b, got)
		}
	}
	if string(got) != "helloworld" {
		t.Fatalf("got %q, want %q", got, "helloworld")
	}
}

// FuzzSanitizeInput: no input may crash the sanitiser or yield a control byte
// or escape byte outside the whitelist. This is the buffer-overflow /
// injection negative test the validation plan requires.
func FuzzSanitizeInput(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte("\x1b[31mx\x1b[0m"))
	f.Add(bytes.Repeat([]byte{0x1b, '['}, 1000))
	f.Add([]byte{0, 1, 2, 27, 91, 255, 254})
	f.Fuzz(func(t *testing.T, data []byte) {
		out := session.SanitizeInput(data)
		if len(out) > len(data) {
			t.Fatalf("output longer than input: %d > %d", len(out), len(data))
		}
		for _, b := range out {
			if b == 0x1B {
				t.Fatalf("escape byte survived sanitise")
			}
			if (b < 0x20 || b == 0x7F) && b != '\t' && b != '\r' && b != '\n' {
				t.Fatalf("disallowed control byte %d survived", b)
			}
		}
	})
}
