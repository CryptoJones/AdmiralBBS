package screen

import (
	"bytes"
	"os"
)

// LoadArt reads a CP437-encoded .ans file. Missing files return (nil, err) so
// the caller can fall back to a plain banner.
func LoadArt(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// RenderArt writes CP437/ANSI art to w. On an ANSI terminal the bytes are sent
// verbatim. On a B&W terminal the ANSI escape sequences are stripped and the
// high CP437 bytes are folded to ASCII approximations so the screen is still
// readable rather than garbled.
func RenderArt(s *Writer, art []byte) {
	if s.ansi {
		s.w.Write(art)
		return
	}
	s.w.Write(degradeCP437(stripANSI(art)))
}

// stripANSI removes ESC-introduced control sequences. CSI sequences (ESC '[')
// are consumed through their final byte (0x40–0x7E); other ESC sequences drop
// the ESC and the following byte.
func stripANSI(in []byte) []byte {
	out := make([]byte, 0, len(in))
	for i := 0; i < len(in); i++ {
		if in[i] == 0x1B {
			if i+1 < len(in) && in[i+1] == '[' {
				i += 2
				for i < len(in) && (in[i] < 0x40 || in[i] > 0x7E) {
					i++
				}
			} else {
				i++
			}
			continue
		}
		out = append(out, in[i])
	}
	return out
}

// cp437Fold maps the CP437 line/box-drawing range to ASCII look-alikes.
func degradeCP437(in []byte) []byte {
	out := make([]byte, 0, len(in))
	for _, b := range in {
		switch {
		case b == '\r' || b == '\n' || b == '\t':
			out = append(out, b)
		case b == 0x1A: // CP437 EOF / SUB marker often trailing .ans files
			// stop at the DOS EOF marker
			return out
		case b < 0x20 || b == 0x7F:
			// drop other control bytes
		case b < 0x80:
			out = append(out, b) // plain ASCII
		default:
			out = append(out, foldHigh(b))
		}
	}
	return out
}

func foldHigh(b byte) byte {
	switch b {
	case 0xB3, 0xBA, 0xB4, 0xB5, 0xB6, 0xB9, 0xC3, 0xC6, 0xC7, 0xCC, 0xD0, 0xD2, 0xD5, 0xD8: // vertical-ish
		return '|'
	case 0xC4, 0xCD, 0xC1, 0xC2, 0xCA, 0xCB, 0xD1, 0xCF: // horizontal-ish
		return '-'
	case 0xDA, 0xBF, 0xC0, 0xD9, 0xC9, 0xBB, 0xC8, 0xBC, 0xC5, 0xCE, 0xD6, 0xB7, 0xD3, 0xBD: // corners/joins
		return '+'
	case 0xB0, 0xB1, 0xB2, 0xDB, 0xDC, 0xDD, 0xDE, 0xDF: // shade/block
		return '#'
	default:
		return '?'
	}
}

// isBlank reports whether art is effectively empty (used by callers).
func isBlank(art []byte) bool { return len(bytes.TrimSpace(art)) == 0 }
