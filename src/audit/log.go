// Package audit records the BBS session audit trail: who connected from where,
// when, what they did, and when they left.
//
// Sprint 002 has no database yet (SQLite arrives in Sprint 003), so events are
// appended to a structured JSONL file and mirrored to stdout. The event shape
// is chosen to migrate cleanly into the session_log / activity tables later
// (see docs/DATA_MODEL.md).
package audit

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Event types.
const (
	TypeConnect    = "connect"
	TypeActivity   = "activity"
	TypeDisconnect = "disconnect"
)

// Event is one line in the audit trail. Fields the operator asked for:
// remote IP, username, connection time, activities, disconnection time.
type Event struct {
	Time      time.Time `json:"time"`
	Type      string    `json:"type"`
	SessionID string    `json:"session_id"`
	RemoteIP  string    `json:"remote_ip"`
	Transport string    `json:"transport"`         // "telnet" | "ssh"
	Username  string    `json:"username,omitempty"` // empty until login (Sprint 003)
	Action    string    `json:"action,omitempty"`   // activity name
	Detail    string    `json:"detail,omitempty"`
	Minutes   float64   `json:"minutes,omitempty"` // session duration on disconnect
}

// Logger appends events to a JSONL file and mirrors a human line to stdout.
// Safe for concurrent use by many caller goroutines.
type Logger struct {
	mu   sync.Mutex
	file io.WriteCloser
	enc  *json.Encoder
	log  *slog.Logger
}

// New opens (or creates) the JSONL audit file at path for appending.
func New(path string) (*Logger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}
	return &Logger{
		file: f,
		enc:  json.NewEncoder(f),
		log:  slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}, nil
}

// Emit writes one event durably to the JSONL file and mirrors it to stdout.
func (l *Logger) Emit(e Event) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.enc.Encode(e); err == nil {
		if f, ok := l.file.(*os.File); ok {
			_ = f.Sync()
		}
	}
	l.log.Info("audit",
		"type", e.Type,
		"session", e.SessionID,
		"ip", e.RemoteIP,
		"transport", e.Transport,
		"user", e.Username,
		"action", e.Action,
		"detail", e.Detail,
		"minutes", e.Minutes,
	)
}

// Close flushes and closes the underlying file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}
