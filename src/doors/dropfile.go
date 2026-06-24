// Package doors launches door games as sandboxed subprocesses (RISKS SEC-1).
package doors

import (
	"fmt"
	"os"
	"strings"
)

// DropInfo is the caller context handed to a door game via the dropfile.
type DropInfo struct {
	BBSName     string
	Handle      string
	RealName    string
	AccessLevel int
	MinutesLeft int
	Node        int
	ANSI        bool
}

// WriteDoor32 writes a standard door32.sys dropfile — the modern, widely-read
// door interchange format (11 lines).
func WriteDoor32(path string, d DropInfo) error {
	emu := "0" // 0 = ASCII
	if d.ANSI {
		emu = "1" // 1 = ANSI
	}
	real := d.RealName
	if real == "" {
		real = d.Handle
	}
	lines := []string{
		"2",                            // comm type: 2 = telnet
		"0",                            // comm/socket handle
		"115200",                       // baud rate
		d.BBSName,                      // BBS name
		"1",                            // user record position
		real,                           // user real name
		d.Handle,                       // user handle/alias
		fmt.Sprintf("%d", d.AccessLevel),
		fmt.Sprintf("%d", d.MinutesLeft),
		emu,                            // terminal emulation
		fmt.Sprintf("%d", d.Node),      // node number
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\r\n")+"\r\n"), 0o600)
}
