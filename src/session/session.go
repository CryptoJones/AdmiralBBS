package session

import (
	"bufio"
	"io"
	"net"
	"sync"
	"time"

	"admiralbbs/src/audit"
	"admiralbbs/src/transport"
)

// Clock lets tests inject time without the real wall clock.
type Clock func() time.Time

// Session is the transport-agnostic caller. Every subsystem is written against
// it. It owns the hardened input reader, the terminal capability, and the
// audit trail for this caller.
type Session struct {
	id    string
	conn  transport.Conn
	log   *audit.Logger
	now   Clock
	start time.Time

	r *bufio.Reader

	mu          sync.Mutex
	cap         Capability
	idle        time.Duration
	idleTimer   *time.Timer
	budgetTimer *time.Timer
}

// New wraps a connection in a session, detects its terminal, logs the connect
// event, and starts tracking live window-size changes.
func New(id string, conn transport.Conn, log *audit.Logger, now Clock) *Session {
	if now == nil {
		now = time.Now
	}
	s := &Session{
		id:    id,
		conn:  conn,
		log:   log,
		now:   now,
		start: now(),
		r:     bufio.NewReader(conn),
		cap:   DetectCapability(conn.TermType(), conn.WindowSize()),
	}

	s.log.Emit(audit.Event{
		Time:      s.start,
		Type:      audit.TypeConnect,
		SessionID: id,
		RemoteIP:  ipOf(conn.RemoteAddr()),
		Transport: conn.Transport(),
		Username:  conn.Username(),
	})

	if ch := conn.WindowChanges(); ch != nil {
		go s.trackResizes(ch)
	}
	return s
}

func (s *Session) trackResizes(ch <-chan transport.WindowSize) {
	for ws := range ch {
		s.mu.Lock()
		if ws.Cols > 0 {
			s.cap.Cols = ws.Cols
		}
		s.cap.Rows = ws.Rows
		s.mu.Unlock()
	}
}

// WatchIdle disconnects the caller after d of no input (SEC-3: slow-loris /
// idle defence). A non-positive d disables the watchdog.
func (s *Session) WatchIdle(d time.Duration) {
	if d <= 0 {
		return
	}
	s.idle = d
	s.idleTimer = time.AfterFunc(d, func() { s.conn.Close() })
}

func (s *Session) resetIdle() {
	if s.idleTimer != nil {
		s.idleTimer.Reset(s.idle)
	}
}

// Cap returns a snapshot of the current terminal capability.
func (s *Session) Cap() Capability {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cap
}

// ID returns the session identifier used in the audit trail.
func (s *Session) ID() string { return s.id }

// Username returns the login name the transport carried (SSH user; "" for telnet).
func (s *Session) Username() string { return s.conn.Username() }

// IP returns the caller's remote host (no port).
func (s *Session) IP() string { return ipOf(s.conn.RemoteAddr()) }

// Transport returns the transport name ("ssh" / "telnet").
func (s *Session) Transport() string { return s.conn.Transport() }

// WatchBudget disconnects the caller after d regardless of activity — the daily
// time-budget cap. A non-positive d disables it.
func (s *Session) WatchBudget(d time.Duration) {
	if d <= 0 {
		return
	}
	s.budgetTimer = time.AfterFunc(d, func() { s.conn.Close() })
}

// Write sends bytes to the caller (Session is an io.Writer).
func (s *Session) Write(p []byte) (int, error) { return s.conn.Write(p) }

// Raw returns a raw byte-level ReadWriter over the connection, for subsystems
// that drive their own I/O (e.g. door games piping to a subprocess). Reads come
// through the session's buffered reader (so no buffered input is lost) and reset
// the idle watchdog; writes go straight to the wire with no sanitisation.
func (s *Session) Raw() io.ReadWriter { return rawIO{s} }

type rawIO struct{ s *Session }

func (x rawIO) Read(p []byte) (int, error) {
	n, err := x.s.r.Read(p)
	if n > 0 {
		x.s.resetIdle()
	}
	return n, err
}

func (x rawIO) Write(p []byte) (int, error) { return x.s.conn.Write(p) }

// Activity records a caller action in the audit trail.
func (s *Session) Activity(action, detail string) {
	s.log.Emit(audit.Event{
		Time:      s.now(),
		Type:      audit.TypeActivity,
		SessionID: s.id,
		RemoteIP:  ipOf(s.conn.RemoteAddr()),
		Transport: s.conn.Transport(),
		Username:  s.conn.Username(),
		Action:    action,
		Detail:    detail,
	})
}

// Close logs the disconnect (with session duration) and closes the wire.
func (s *Session) Close() error {
	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}
	if s.budgetTimer != nil {
		s.budgetTimer.Stop()
	}
	end := s.now()
	s.log.Emit(audit.Event{
		Time:      end,
		Type:      audit.TypeDisconnect,
		SessionID: s.id,
		RemoteIP:  ipOf(s.conn.RemoteAddr()),
		Transport: s.conn.Transport(),
		Username:  s.conn.Username(),
		Minutes:   end.Sub(s.start).Minutes(),
	})
	return s.conn.Close()
}

// nextByte returns the next caller byte with escape sequences consumed and
// discarded. It is the single chokepoint through which all input passes.
func (s *Session) nextByte() (byte, error) {
	for {
		b, err := s.r.ReadByte()
		if err != nil {
			return 0, err
		}
		s.resetIdle() // input arrived: postpone the idle watchdog (SEC-3)
		if b != 0x1B { // not ESC: a real byte
			return b, nil
		}
		// ESC: consume the rest of the sequence and ignore it.
		nb, err := s.r.ReadByte()
		if err != nil {
			return 0, err
		}
		if nb == '[' {
			for {
				fb, err := s.r.ReadByte()
				if err != nil {
					return 0, err
				}
				if fb >= 0x40 && fb <= 0x7E {
					break // CSI final byte
				}
			}
		}
		// loop to fetch the next real byte
	}
}

// ReadKey reads a single sanitised keypress (no echo). Disallowed control
// bytes are skipped. Returns the byte, or an error on disconnect.
func (s *Session) ReadKey() (byte, error) {
	for {
		b, err := s.nextByte()
		if err != nil {
			return 0, err
		}
		if b < 0x20 || b == 0x7F {
			if b == '\r' || b == '\n' {
				return b, nil
			}
			continue // drop other control bytes
		}
		return b, nil
	}
}

// ReadLine reads a sanitised, length-bounded line (terminated by CR or LF),
// echoing printable input and handling backspace. The result never exceeds
// MaxLineLen and never contains control or escape bytes.
func (s *Session) ReadLine() (string, error) {
	buf := make([]byte, 0, 64)
	for {
		b, err := s.nextByte()
		if err != nil {
			return "", err
		}
		switch {
		case b == '\r' || b == '\n':
			io.WriteString(s.conn, "\r\n")
			return string(buf), nil
		case b == 0x08 || b == 0x7F: // backspace / delete
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				io.WriteString(s.conn, "\b \b")
			}
		case b < 0x20:
			// drop other control bytes
		default:
			if len(buf) < MaxLineLen {
				buf = append(buf, b)
				s.conn.Write([]byte{b}) // echo
			}
		}
	}
}

// ReadPassword reads a line WITHOUT echoing it — for password entry. Bounded
// and sanitised like ReadLine, but nothing is written back to the terminal.
func (s *Session) ReadPassword() (string, error) {
	buf := make([]byte, 0, 32)
	for {
		b, err := s.nextByte()
		if err != nil {
			return "", err
		}
		switch {
		case b == '\r' || b == '\n':
			io.WriteString(s.conn, "\r\n")
			return string(buf), nil
		case b == 0x08 || b == 0x7F:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
		case b < 0x20:
			// drop
		default:
			if len(buf) < MaxLineLen {
				buf = append(buf, b)
			}
		}
	}
}

func ipOf(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	if host, _, err := net.SplitHostPort(addr.String()); err == nil {
		return host
	}
	return addr.String()
}
