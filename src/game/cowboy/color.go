package cowboy

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

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

// banner is shown on connect. The box is sized to its widest content line and
// each row is padded by rune count (so multi-byte glyphs like the em-dash don't
// throw off the right border) — keeping ╔/║/╝ vertically aligned on any line.
func banner() string {
	title := "  C H R O M E   C I R C U I T   C O W B O Y S   [ C³ ]"
	tag := "  a cyberpunk netrun — jack in, level up, breach the ICE  "
	inner := utf8.RuneCountInString(tag)
	if w := utf8.RuneCountInString(title); w > inner {
		inner = w
	}
	bar := strings.Repeat("═", inner)
	row := func(text, color string) string {
		pad := inner - utf8.RuneCountInString(text)
		if pad < 0 {
			pad = 0
		}
		return style(neon, "║") + style(color, text+strings.Repeat(" ", pad)) + style(neon, "║") + crlf
	}
	return crlf +
		style(neon, "╔"+bar+"╗") + crlf +
		row(title, hot) +
		row(tag, dim) +
		style(neon, "╚"+bar+"╝") + crlf +
		style(dim, "Type HELP for commands. Movement: N S E W U D.") + crlf
}
