package transport

import (
	"net"
	"testing"
	"time"
)

// discardConn is a no-op net.Conn so the telnet parser's negotiation replies
// (IAC WONT/DONT, etc.) have somewhere to go during fuzzing.
type discardConn struct{}

func (discardConn) Read(b []byte) (int, error)         { return 0, net.ErrClosed }
func (discardConn) Write(b []byte) (int, error)        { return len(b), nil }
func (discardConn) Close() error                       { return nil }
func (discardConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (discardConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (discardConn) SetDeadline(t time.Time) error      { return nil }
func (discardConn) SetReadDeadline(t time.Time) error  { return nil }
func (discardConn) SetWriteDeadline(t time.Time) error { return nil }

// FuzzTelnetFeed throws arbitrary bytes at the telnet IAC state machine; it must
// never panic, and decoded data must stay bounded by the input.
func FuzzTelnetFeed(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte{iac, do, optNAWS})
	f.Add([]byte{iac, sb, optNAWS, 0, 80, 0, 24, iac, se})
	f.Add([]byte{iac, sb, optTType, ttypeIS, 'a', 'n', 's', 'i', iac, se})
	f.Add([]byte{iac, iac, iac, will, 1, iac})
	f.Fuzz(func(t *testing.T, data []byte) {
		c := &telnetConn{raw: discardConn{}, winCh: make(chan WindowSize, 8)}
		c.feed(data) // must not panic on any input
		if len(c.pending) > len(data) {
			t.Fatalf("decoded %d bytes from %d input bytes", len(c.pending), len(data))
		}
	})
}
