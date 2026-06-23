// Package transport hosts the two listeners (Telnet and SSH) and the single
// contract — Conn — that both satisfy. Everything above this layer (session,
// menu) is written against Conn and never learns which wire it is on.
package transport

import (
	"io"
	"net"
)

// WindowSize is the caller's terminal dimensions. Per the ANSI-BBS spec we may
// assume 80 columns when unknown, but must never assume a row count for layout.
type WindowSize struct {
	Cols int
	Rows int
}

// Conn is a transport-agnostic caller connection. Telnet and SSH each provide
// an implementation; the session layer depends only on this interface.
type Conn interface {
	io.ReadWriteCloser

	// RemoteAddr is the caller's network address (for the audit IP).
	RemoteAddr() net.Addr

	// Transport identifies the wire: "telnet" or "ssh".
	Transport() string

	// TermType is the negotiated terminal type (e.g. "ansi", "xterm",
	// "syncterm"), or "" if the caller never told us.
	TermType() string

	// WindowSize is the most recently known terminal size.
	WindowSize() WindowSize

	// WindowChanges streams live resize events (SSH window-change / Telnet
	// NAWS). May be nil if the transport does not report resizes.
	WindowChanges() <-chan WindowSize

	// Username is the login name supplied at connect time, if any. Telnet
	// supplies none pre-account; SSH carries the ssh user. "" when unknown.
	Username() string
}
