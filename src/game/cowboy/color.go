package cowboy

import "strconv"

// crlf is the line terminator used on the wire (telnet/SSH terminals).
const crlf = "\r\n"

// ANSI SGR codes for the neon palette. Bytes are passed straight through the
// BBS bridge to the caller's terminal.
const (
	reset   = "\x1b[0m"
	neon    = "\x1b[1;36m" // bright cyan — system/headers
	hot     = "\x1b[1;35m" // bright magenta — combat/alerts
	gold    = "\x1b[1;33m" // yellow — currency/rewards
	green   = "\x1b[1;32m" // green — prompts/success
	dim     = "\x1b[0;90m" // grey — ambience
	red     = "\x1b[1;31m" // red — damage/danger
)

// style wraps s in an SGR color and a reset.
func style(code, s string) string { return code + s + reset }

func itoa(n int) string { return strconv.Itoa(n) }

// banner is shown on connect.
func banner() string {
	return crlf +
		style(neon, "╔══════════════════════════════════════════════════════╗") + crlf +
		style(neon, "║  ") + style(hot, "C O N S O L E   C O W B O Y   2 0 2 6") + style(neon, "             ║") + crlf +
		style(neon, "║  ") + style(dim, "a cyberpunk netrun — jack in, level up, breach the ICE") + style(neon, "║") + crlf +
		style(neon, "╚══════════════════════════════════════════════════════╝") + crlf +
		style(dim, "Type HELP for commands. Movement: N S E W U D.") + crlf
}
