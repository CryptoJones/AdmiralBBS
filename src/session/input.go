package session

// Hardened input handling. This is the security boundary named in
// DECISIONS.md: caller input is length-bounded and sanitised before it reaches
// any parser or is echoed back to a terminal. A memory-safe language already
// removes the buffer-overflow class; this layer removes the
// control-character / escape-sequence injection class ("packet injection").

// MaxLineLen bounds a single line read so a hostile caller cannot make the
// server allocate without limit.
const MaxLineLen = 4096

// allowedControl is the small set of control bytes we let through unchanged.
func allowedControl(b byte) bool {
	switch b {
	case '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

// SanitizeInput strips dangerous bytes from a caller-supplied buffer:
//   - control characters (0x00–0x1F, 0x7F) except TAB/CR/LF are dropped;
//   - ANSI/VT escape sequences are removed entirely — an ESC followed by '['
//     (CSI) is consumed through its final byte (0x40–0x7E); an ESC followed by
//     any other byte drops both (the ANSI-BBS "legal function" range);
//   - printable bytes (including high CP437 bytes 0x80–0xFF) pass through.
//
// It is a pure function so it can be unit-tested and fuzzed directly.
func SanitizeInput(in []byte) []byte {
	out := make([]byte, 0, len(in))
	for i := 0; i < len(in); i++ {
		b := in[i]
		switch {
		case b == 0x1B: // ESC — start of an escape sequence
			if i+1 < len(in) && in[i+1] == '[' {
				// CSI: skip until a final byte in 0x40–0x7E.
				i += 2
				for i < len(in) && (in[i] < 0x40 || in[i] > 0x7E) {
					i++
				}
				// i now points at the final byte (or end); loop's i++ consumes it.
			} else {
				// ESC + single function char: drop both.
				i++
			}
		case b < 0x20 || b == 0x7F:
			if allowedControl(b) {
				out = append(out, b)
			}
			// else: drop the control byte.
		default:
			out = append(out, b) // printable ASCII or high CP437 byte
		}
	}
	return out
}
