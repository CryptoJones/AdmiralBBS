package store

import (
	"log"

	"admiralbbs/src/audit"
)

// SessionLog mirrors the authoritative (encrypted, hash-chained) JSONL audit
// trail into SQLite for SysOp queryability. It implements audit.Sink. Insert
// failures are logged but never propagated — the JSONL file is the source of
// truth, so a DB hiccup must not break a caller's session.
//
// Structural columns (session_id, ip, transport, username, event_type, at) stay
// queryable; the free-text `detail` is encrypted (the app-level layer). The
// encrypted volume covers the structural columns offline.
type SessionLog struct{ st *Store }

// Emit writes one audit event to the session_log table (best-effort).
func (sl *SessionLog) Emit(e audit.Event) {
	detail, err := sl.st.seal(e.Detail)
	if err != nil {
		log.Printf("session_log: seal detail failed: %v", err)
		return
	}
	_, err = sl.st.db.Exec(
		`INSERT INTO session_log
		   (session_id, username, transport, remote_ip, event_type, action, detail, minutes, at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.SessionID, e.Username, e.Transport, e.RemoteIP, e.Type, e.Action, detail, e.Minutes,
		e.Time.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	)
	if err != nil {
		log.Printf("session_log mirror insert failed (JSONL trail unaffected): %v", err)
	}
}

// Count returns the number of session_log rows (used by tests / SysOp tools).
func (sl *SessionLog) Count() (int, error) {
	var n int
	err := sl.st.db.QueryRow(`SELECT COUNT(*) FROM session_log`).Scan(&n)
	return n, err
}
