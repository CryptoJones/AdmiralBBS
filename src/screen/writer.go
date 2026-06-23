// Package screen renders output to a caller. It has exactly one render path:
// when the terminal is ANSI-capable it emits colour and cursor-control codes;
// otherwise the same calls degrade to plain black-and-white text. Callers do
// not branch on capability — the Writer does (DECISIONS.md).
package screen

import (
	"fmt"
	"io"
)

// ANSI colour codes (foreground).
const (
	Black = iota + 30
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

// Writer wraps an output stream with capability-aware rendering.
type Writer struct {
	w    io.Writer
	ansi bool
	cols int
}

// New builds a Writer. ansi=false produces plain B&W output from every call.
func New(w io.Writer, ansi bool, cols int) *Writer {
	if cols <= 0 {
		cols = 80
	}
	return &Writer{w: w, ansi: ansi, cols: cols}
}

// Cols reports the assumed line width.
func (s *Writer) Cols() int { return s.cols }

// Print writes text verbatim.
func (s *Writer) Print(text string) { io.WriteString(s.w, text) }

// Printf writes formatted text verbatim.
func (s *Writer) Printf(format string, a ...any) {
	fmt.Fprintf(s.w, format, a...)
}

// Line writes text followed by CRLF (BBS line ending).
func (s *Writer) Line(text string) { io.WriteString(s.w, text+"\r\n") }

// Color sets the foreground colour — a no-op on B&W terminals.
func (s *Writer) Color(fg int) {
	if s.ansi {
		fmt.Fprintf(s.w, "\x1b[%dm", fg)
	}
}

// Bold enables bold/bright — a no-op on B&W terminals.
func (s *Writer) Bold() {
	if s.ansi {
		io.WriteString(s.w, "\x1b[1m")
	}
}

// Reset clears attributes — a no-op on B&W terminals.
func (s *Writer) Reset() {
	if s.ansi {
		io.WriteString(s.w, "\x1b[0m")
	}
}

// Clear clears the screen. ANSI homes the cursor and erases; B&W emits a
// blank separator so output does not pile up unreadably.
func (s *Writer) Clear() {
	if s.ansi {
		io.WriteString(s.w, "\x1b[2J\x1b[H")
	} else {
		io.WriteString(s.w, "\r\n\r\n")
	}
}

// ColorLine writes a coloured line that resets afterward (no-op colour on B&W).
func (s *Writer) ColorLine(fg int, text string) {
	s.Color(fg)
	s.Print(text)
	s.Reset()
	s.Print("\r\n")
}

// SafePrint writes user-generated content with escape sequences and control
// bytes stripped, so one caller cannot inject terminal-hijacking sequences into
// another caller's screen (RISKS SEC-5: sanitise on OUTPUT, not just input).
func (s *Writer) SafePrint(text string) { s.Print(SanitizeForDisplay(text)) }

// SanitizeForDisplay removes ANSI/VT escape sequences and C0/DEL control bytes
// from text destined for a terminal, keeping printable ASCII, high CP437 bytes,
// and ordinary spacing (space, tab, CR, LF).
func SanitizeForDisplay(text string) string {
	in := []byte(text)
	out := make([]byte, 0, len(in))
	for i := 0; i < len(in); i++ {
		b := in[i]
		switch {
		case b == 0x1B: // ESC: drop the whole sequence
			if i+1 < len(in) && in[i+1] == '[' {
				i += 2
				for i < len(in) && (in[i] < 0x40 || in[i] > 0x7E) {
					i++
				}
			} else {
				i++
			}
		case b == '\t' || b == '\r' || b == '\n' || b == ' ':
			out = append(out, b)
		case b < 0x20 || b == 0x7F:
			// drop other control bytes
		default:
			out = append(out, b)
		}
	}
	return string(out)
}
