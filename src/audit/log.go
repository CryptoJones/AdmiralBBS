// Package audit records the BBS session audit trail: who connected from where,
// when, what they did, and when they left.
//
// The on-disk trail is BOTH confidential and tamper-evident:
//   - each event's JSON is sealed with the crypto.Vault (XChaCha20-Poly1305),
//     so the file is unreadable without the key;
//   - each line carries an HMAC that commits to the previous line's HMAC,
//     forming a hash-chain — any edit, deletion, or truncation is detectable
//     (see VerifyChain). Confidentiality + integrity (RISKS SEC-6).
//
// Line format: base64url(ciphertext) "." base64url(mac).
// The JSONL-style file is authoritative; sinks (e.g. the SQLite session_log)
// are best-effort mirrors.
package audit

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"admiralbbs/src/crypto"
)

// Event types.
const (
	TypeConnect    = "connect"
	TypeActivity   = "activity"
	TypeDisconnect = "disconnect"
)

var genesis = []byte("admiralbbs/audit-genesis/v1")

// Event is one entry in the audit trail.
type Event struct {
	Time      time.Time `json:"time"`
	Type      string    `json:"type"`
	SessionID string    `json:"session_id"`
	RemoteIP  string    `json:"remote_ip"`
	Transport string    `json:"transport"`
	Username  string    `json:"username,omitempty"`
	Action    string    `json:"action,omitempty"`
	Detail    string    `json:"detail,omitempty"`
	Minutes   float64   `json:"minutes,omitempty"`
}

// Sink receives every event after the authoritative encrypted write. Sinks are
// best-effort mirrors and must not block or panic.
type Sink interface{ Emit(Event) }

// Logger seals events to an append-only hash-chained file, mirrors a redacted
// line to stdout, and fans out to sinks. Safe for concurrent use.
type Logger struct {
	mu      sync.Mutex
	file    io.WriteCloser
	vault   *crypto.Vault
	prevMAC []byte
	log     *slog.Logger
	sinks   []Sink
}

var b64 = base64.RawURLEncoding

// New opens (creating if needed) the encrypted audit file at path and recovers
// the hash-chain tip so appends continue the chain. A vault is required.
func New(path string, vault *crypto.Vault, sinks ...Sink) (*Logger, error) {
	if vault == nil {
		return nil, errors.New("audit: vault is required (encryption is mandatory)")
	}
	tip, err := chainTip(path, vault)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &Logger{
		file:    f,
		vault:   vault,
		prevMAC: tip,
		log:     slog.New(slog.NewTextHandler(os.Stdout, nil)),
		sinks:   sinks,
	}, nil
}

// chainTip returns the MAC of the last line in path (or the genesis MAC if the
// file is empty/absent), so a new Logger can extend the chain.
func chainTip(path string, vault *crypto.Vault) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && len(bytes.TrimSpace(data)) == 0) {
		return vault.MAC(nil, genesis), nil
	}
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	last := lines[len(lines)-1]
	_, mac, err := splitLine(last)
	if err != nil {
		return nil, err
	}
	return mac, nil
}

// Emit seals the event, appends it to the chain, mirrors a redacted line to
// stdout, and fans out to sinks.
func (l *Logger) Emit(e Event) {
	plain, err := json.Marshal(e)
	if err != nil {
		return
	}
	l.mu.Lock()
	ct, err := l.vault.Seal(plain)
	if err == nil {
		mac := l.vault.MAC(l.prevMAC, ct)
		if _, werr := io.WriteString(l.file, b64.EncodeToString(ct)+"."+b64.EncodeToString(mac)+"\n"); werr == nil {
			l.prevMAC = mac
			if f, ok := l.file.(*os.File); ok {
				_ = f.Sync()
			}
		}
	}
	sinks := l.sinks
	l.mu.Unlock()

	for _, s := range sinks {
		s.Emit(e)
	}

	// Stdout mirror is operational telemetry — redact free-text detail.
	l.log.Info("audit",
		"type", e.Type, "session", e.SessionID, "ip", e.RemoteIP,
		"transport", e.Transport, "user", e.Username, "action", e.Action,
		"minutes", e.Minutes)
}

// Close closes the underlying file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

func splitLine(line string) (ct, mac []byte, err error) {
	i := strings.IndexByte(line, '.')
	if i <= 0 {
		return nil, nil, errors.New("audit: malformed line")
	}
	ct, err = b64.DecodeString(line[:i])
	if err != nil {
		return nil, nil, err
	}
	mac, err = b64.DecodeString(line[i+1:])
	if err != nil {
		return nil, nil, err
	}
	return ct, mac, nil
}

// ErrChainBroken reports that the hash-chain failed to verify — the trail was
// edited, truncated, or reordered.
var ErrChainBroken = errors.New("audit: hash-chain verification failed (tampering detected)")

// ReadAll verifies the hash-chain end-to-end and returns the decrypted events.
// A broken chain returns ErrChainBroken.
func ReadAll(path string, vault *crypto.Vault) ([]Event, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	prev := vault.MAC(nil, genesis)
	var events []Event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		ct, mac, err := splitLine(line)
		if err != nil {
			return nil, err
		}
		want := vault.MAC(prev, ct)
		if !hmacEqual(want, mac) {
			return events, ErrChainBroken
		}
		plain, err := vault.Open(ct)
		if err != nil {
			return events, ErrChainBroken
		}
		var e Event
		if err := json.Unmarshal(plain, &e); err != nil {
			return nil, err
		}
		events = append(events, e)
		prev = mac
	}
	return events, sc.Err()
}

func hmacEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
