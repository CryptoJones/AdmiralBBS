package transport

import (
	"net"
	"strings"
	"sync"
)

// Telnet command + option bytes (RFC 854 / 1073 / 1091).
const (
	iac  = 255
	se   = 240
	sb   = 250
	will = 251
	wont = 252
	do   = 253
	dont = 254

	optEcho  = 1
	optSGA   = 3
	optTType = 24
	optNAWS  = 31

	ttypeIS   = 0
	ttypeSEND = 1
)

// ServeTelnet listens on addr and hands each accepted caller (as a Conn) to
// handle in its own goroutine, subject to the session/per-IP caps in limits.
// Plaintext by design — authenticity (and Telnet is apply-only).
func ServeTelnet(addr string, limits Limits, banned BanCheck, handle func(Conn)) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	lm := newLimiter(limits)
	for {
		raw, err := ln.Accept()
		if err != nil {
			return err
		}
		ip := hostOf(raw.RemoteAddr())
		if banned != nil && banned(ip) {
			raw.Close() // banned source — drop before anything else
			continue
		}
		if !lm.acquire(ip) {
			raw.Close() // over a cap; drop quietly
			continue
		}
		go func(raw net.Conn, ip string) {
			defer lm.release(ip)
			handle(newTelnetConn(raw))
		}(raw, ip)
	}
}

// telnet parser states.
const (
	stData = iota
	stIAC
	stOpt // expecting the option byte after WILL/WONT/DO/DONT
	stSB  // inside subnegotiation, collecting data
	stSBI // inside subnegotiation, saw IAC (awaiting SE or escaped IAC)
)

type telnetConn struct {
	raw net.Conn

	mu       sync.Mutex
	termType string
	win      WindowSize

	winCh chan WindowSize

	// parser state (only touched by Read's goroutine)
	state   int
	cmd     byte
	subOpt  byte
	subData []byte
	pending []byte // decoded data bytes not yet returned by Read
}

func newTelnetConn(raw net.Conn) *telnetConn {
	c := &telnetConn{raw: raw, winCh: make(chan WindowSize, 8)}
	// Proactive negotiation: we echo, suppress go-ahead, and ask the client
	// for its window size and terminal type.
	c.raw.Write([]byte{
		iac, will, optEcho,
		iac, will, optSGA,
		iac, do, optNAWS,
		iac, do, optTType,
		iac, sb, optTType, ttypeSEND, iac, se,
	})
	return c
}

func (c *telnetConn) Read(p []byte) (int, error) {
	for len(c.pending) == 0 {
		buf := make([]byte, 1024)
		n, err := c.raw.Read(buf)
		if n > 0 {
			c.feed(buf[:n])
		}
		if err != nil {
			if len(c.pending) > 0 {
				break
			}
			return 0, err
		}
	}
	n := copy(p, c.pending)
	c.pending = c.pending[n:]
	return n, nil
}

// feed runs the telnet state machine over a raw chunk, appending decoded data
// bytes to c.pending and handling negotiation / subnegotiation inline.
func (c *telnetConn) feed(raw []byte) {
	for _, b := range raw {
		switch c.state {
		case stData:
			if b == iac {
				c.state = stIAC
			} else {
				c.pending = append(c.pending, b)
			}
		case stIAC:
			switch b {
			case iac: // escaped 255 → literal data byte
				c.pending = append(c.pending, iac)
				c.state = stData
			case sb:
				c.state = stSB
				c.subOpt = 0
				c.subData = c.subData[:0]
			case will, wont, do, dont:
				c.cmd = b
				c.state = stOpt
			default:
				c.state = stData // standalone command (GA, NOP, ...) ignored
			}
		case stOpt:
			c.respond(c.cmd, b)
			c.state = stData
		case stSB:
			if c.subOpt == 0 && b != iac {
				c.subOpt = b
				continue
			}
			if b == iac {
				c.state = stSBI
			} else {
				c.subData = append(c.subData, b)
			}
		case stSBI:
			if b == se {
				c.endSubneg()
				c.state = stData
			} else { // escaped IAC inside subneg data
				c.subData = append(c.subData, b)
				c.state = stSB
			}
		}
	}
}

// respond replies to a WILL/WONT/DO/DONT to keep negotiation from looping.
// Minimal policy: accept the few options we use, refuse the rest.
func (c *telnetConn) respond(cmd, opt byte) {
	switch cmd {
	case do:
		if opt == optEcho || opt == optSGA {
			return // already advertised WILL for these
		}
		c.raw.Write([]byte{iac, wont, opt})
	case dont:
		c.raw.Write([]byte{iac, wont, opt})
	case will:
		if opt == optTType || opt == optNAWS {
			return // we asked for these
		}
		c.raw.Write([]byte{iac, dont, opt})
	case wont:
		c.raw.Write([]byte{iac, dont, opt})
	}
}

func (c *telnetConn) endSubneg() {
	switch c.subOpt {
	case optNAWS:
		if len(c.subData) >= 4 {
			cols := int(c.subData[0])<<8 | int(c.subData[1])
			rows := int(c.subData[2])<<8 | int(c.subData[3])
			c.mu.Lock()
			c.win = WindowSize{Cols: cols, Rows: rows}
			c.mu.Unlock()
			select {
			case c.winCh <- WindowSize{Cols: cols, Rows: rows}:
			default:
			}
		}
	case optTType:
		if len(c.subData) >= 1 && c.subData[0] == ttypeIS {
			t := strings.TrimSpace(string(c.subData[1:]))
			c.mu.Lock()
			if c.termType == "" {
				c.termType = t
			}
			c.mu.Unlock()
		}
	}
}

func (c *telnetConn) Write(p []byte) (int, error) { return c.raw.Write(p) }
func (c *telnetConn) Close() error                { return c.raw.Close() }
func (c *telnetConn) RemoteAddr() net.Addr        { return c.raw.RemoteAddr() }
func (c *telnetConn) Transport() string           { return "telnet" }

func (c *telnetConn) TermType() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.termType
}

func (c *telnetConn) WindowSize() WindowSize {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.win
}

func (c *telnetConn) WindowChanges() <-chan WindowSize { return c.winCh }
func (c *telnetConn) Username() string                 { return "" } // no account pre-login
