package cowboy

import (
	"bufio"
	"io"
)

// ReadLine reads one line of raw terminal input, handling the realities of a
// bridged SSH/telnet caller in raw mode: CR or LF (and CRLF) end the line,
// backspace/DEL erase the last rune, NUL is ignored, and a telnet IAC byte plus
// its command byte are skipped. If echo is non-nil, typed characters are echoed
// back (SSH raw mode doesn't echo locally), including a destructive backspace.
func ReadLine(r *bufio.Reader, echo func(string)) (string, error) {
	var buf []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			if len(buf) > 0 && err == io.EOF {
				return string(buf), nil
			}
			return "", err
		}
		switch b {
		case '\r', '\n':
			// Swallow a paired CRLF/LFCR ONLY if the partner byte is already
			// buffered. Peeking with a blocking ReadByte() here is the bug it
			// replaces: on an interactive lone CR/LF (one keystroke), the peek
			// would block waiting for a byte that never comes, or strand a
			// half-pair that the next read returns as a spurious blank line ->
			// an extra prompt. Buffered() never blocks.
			if r.Buffered() > 0 {
				if next, e := r.ReadByte(); e == nil {
					if !((b == '\r' && next == '\n') || (b == '\n' && next == '\r')) {
						_ = r.UnreadByte()
					}
				}
			}
			if echo != nil {
				echo("\r\n")
			}
			return string(buf), nil
		case 0x08, 0x7f: // backspace / DEL
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				if echo != nil {
					echo("\b \b")
				}
			}
		case 0x00:
			// ignore
		case 0xff: // telnet IAC — skip it and its command byte
			_, _ = r.ReadByte()
		default:
			if b >= 0x20 && b < 0x7f {
				buf = append(buf, b)
				if echo != nil {
					echo(string(b))
				}
			}
		}
	}
}
