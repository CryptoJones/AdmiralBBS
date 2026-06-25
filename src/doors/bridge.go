package doors

import (
	"bytes"
	"io"
	"net"
	"strings"
	"time"
)

// Resident-door handshake (OPTIONAL, generic — nothing game-specific).
//
// As the very first bytes on an accepted bridge connection, a resident door MAY
// advertise itself with an OSC-framed, semicolon-separated key=value string:
//
//	ESC ] ABBS;version=<semver>;caps=<cap,cap,...> BEL
//
// The host strips it from the stream, MAY display the version (e.g. on the
// launch line), and — for each advertised capability — MAY respond. The only
// capability today is "handle": a door that advertises it gets the caller's BBS
// handle pushed back as a reciprocal sentinel (see SendHandle), so it can
// default its own name prompt. Doors that send nothing are fully transparent.
// OSC framing means the sequence is swallowed by a terminal if a door is reached
// directly (not through the BBS), so it never garbles a raw session.
const (
	sentinelPrefix = "\x1b]ABBS;"
	sentinelBEL    = 0x07
	sentinelMax    = 256 // cap the scan so a garbled/hostile door can't make us read forever
)

// ResidentConn is a dialed resident-door connection plus what it advertised
// during the handshake. Relay pumps bytes both ways; Close ends it.
type ResidentConn struct {
	conn    net.Conn
	Version string          // advertised door version (no leading "v"), or "" if none
	caps    map[string]bool // advertised capabilities (e.g. "handle")
	pre     []byte          // door bytes already read past the handshake — replayed to the caller first
}

// DialResident connects to a resident door and reads its OPTIONAL version
// handshake before any game I/O reaches the caller. The handshake wait is
// bounded by handshakeTimeout (a door that sends nothing, or no sentinel, just
// yields Version=="" with everything read replayed verbatim). It never fails the
// connection on a missing/garbled handshake.
func DialResident(network, address string, dialTimeout, handshakeTimeout time.Duration) (*ResidentConn, error) {
	if dialTimeout <= 0 {
		dialTimeout = 10 * time.Second
	}
	conn, err := net.DialTimeout(network, address, dialTimeout)
	if err != nil {
		return nil, err
	}
	rc := &ResidentConn{conn: conn}
	rc.readHandshake(handshakeTimeout)
	return rc, nil
}

// readHandshake peeks the door's first bytes for the version sentinel. On any
// timeout/error it leaves Version=="" and preserves whatever was read as pre, so
// a non-advertising door is transparent.
func (rc *ResidentConn) readHandshake(timeout time.Duration) {
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	_ = rc.conn.SetReadDeadline(time.Now().Add(timeout))
	defer rc.conn.SetReadDeadline(time.Time{}) // restore blocking I/O for the relay

	prefix := []byte(sentinelPrefix)
	buf := make([]byte, 0, sentinelMax)
	tmp := make([]byte, sentinelMax)
	for len(buf) < sentinelMax {
		n, err := rc.conn.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			switch {
			case bytes.HasPrefix(buf, prefix):
				// Full prefix matched — look for the BEL terminator.
				if i := bytes.IndexByte(buf, sentinelBEL); i >= 0 {
					rc.parsePayload(string(buf[len(prefix):i]))
					rc.pre = buf[i+1:]
					return
				}
				// prefix present but no BEL yet — keep reading
			case bytes.HasPrefix(prefix, buf):
				// buf is still a partial prefix of the sentinel — keep reading
			default:
				// Diverged from the sentinel: not a handshake. Forward verbatim.
				rc.pre = buf
				return
			}
		}
		if err != nil {
			rc.pre = buf // timeout or EOF mid-handshake: forward what we have
			return
		}
	}
	rc.pre = buf // overflowed the scan cap without a complete sentinel — forward verbatim
}

// parsePayload reads the handshake's "key=value;key=value" body for the fields
// the host understands (version, caps). Unknown keys are ignored.
func (rc *ResidentConn) parsePayload(s string) {
	rc.caps = map[string]bool{}
	for _, kv := range strings.Split(s, ";") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "version":
			rc.Version = sanitizeVersion([]byte(v))
		case "caps":
			for _, c := range strings.Split(v, ",") {
				if c = strings.TrimSpace(c); c != "" {
					rc.caps[c] = true
				}
			}
		}
	}
}

// SendHandle pushes the caller's BBS handle to the door — but ONLY if the door
// advertised the "handle" capability — so the door can default its own name
// prompt to it. It's the reciprocal OSC sentinel the door reads and strips:
//
//	ESC ] ABBS;handle=<handle> BEL
//
// A no-op for doors that didn't advertise the capability, so it never injects
// bytes a door isn't expecting.
func (rc *ResidentConn) SendHandle(handle string) {
	if rc == nil || !rc.caps["handle"] {
		return
	}
	h := sanitizeHandle(handle)
	if h == "" {
		return
	}
	_, _ = rc.conn.Write([]byte(sentinelPrefix + "handle=" + h + string(rune(sentinelBEL))))
}

// sanitizeHandle keeps only characters legal in a runner/BBS handle and caps the
// length, so the reciprocal sentinel can never carry control codes.
func sanitizeHandle(s string) string {
	if len(s) > 24 {
		s = s[:24]
	}
	out := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		if c == '_' || c == '-' || c == '.' ||
			(c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			out = append(out, c)
		}
	}
	return string(out)
}

// sanitizeVersion keeps only characters legal in a SemVer-ish string and caps the
// length, so a door can never inject control codes into the caller's terminal via
// the handshake.
func sanitizeVersion(b []byte) string {
	if len(b) > 32 {
		b = b[:32]
	}
	out := make([]byte, 0, len(b))
	for _, c := range b {
		if c == '.' || c == '-' || c == '+' ||
			(c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			out = append(out, c)
		}
	}
	return string(out)
}

// Relay replays any bytes read during the handshake, then pumps bytes both ways
// until either side closes, so every caller shares the one live game world.
func (rc *ResidentConn) Relay(sess io.ReadWriter) error {
	if len(rc.pre) > 0 {
		if _, err := sess.Write(rc.pre); err != nil {
			return err
		}
		rc.pre = nil
	}
	done := make(chan struct{}, 2)
	go func() { io.Copy(rc.conn, sess); done <- struct{}{} }() // caller -> game
	go func() { io.Copy(sess, rc.conn); done <- struct{}{} }() // game -> caller
	<-done                                                     // one side hung up
	return nil
}

// Close tears down the door connection.
func (rc *ResidentConn) Close() error { return rc.conn.Close() }

// Bridge connects a caller's session to a RESIDENT door — a persistent,
// real-time multiplayer game server (MajorMUD / Worldgroup style) already
// running and listening at network+address. It relays bytes both ways until
// either side closes. This is the version-agnostic convenience wrapper; callers
// that want the advertised version use DialResident directly.
func Bridge(sess io.ReadWriter, network, address string, dialTimeout time.Duration) error {
	rc, err := DialResident(network, address, dialTimeout, 0)
	if err != nil {
		return err
	}
	defer rc.Close()
	return rc.Relay(sess)
}
