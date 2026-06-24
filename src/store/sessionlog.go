package store

import (
	"database/sql"
	"log"
	"time"

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

// Recent returns the latest audit events from the session_log mirror (newest
// first), decrypting the free-text detail — for the SysOp audit viewer.
func (sl *SessionLog) Recent(limit int) ([]audit.Event, error) {
	return sl.Page(limit, 0)
}

// Page returns a window of audit events (newest first), skipping the newest
// `offset` rows and returning up to `limit`. Backs the SysOp paged viewer's
// next/previous/jump navigation. offset/limit below 0 are clamped to 0.
func (sl *SessionLog) Page(limit, offset int) ([]audit.Event, error) {
	if limit < 0 {
		limit = 0
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := sl.st.db.Query(
		`SELECT session_id, username, transport, remote_ip, event_type, action, detail, minutes, at
		 FROM session_log ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []audit.Event
	for rows.Next() {
		var e audit.Event
		var encDetail, at string
		if err := rows.Scan(&e.SessionID, &e.Username, &e.Transport, &e.RemoteIP,
			&e.Type, &e.Action, &encDetail, &e.Minutes, &at); err != nil {
			return nil, err
		}
		if e.Detail, err = sl.st.open(encDetail); err != nil {
			return nil, err
		}
		e.Time = parseTime(at)
		out = append(out, e)
	}
	return out, rows.Err()
}

// VerifyAuditChain verifies the authoritative JSONL trail end-to-end and
// returns the event count; a non-nil error means tampering was detected.
func (s *Store) VerifyAuditChain(path string) (int, error) {
	events, err := audit.ReadAll(path, s.vault)
	return len(events), err
}

// MinutesToday sums disconnect-event durations for a user since UTC midnight —
// the basis for the daily time budget.
func (sl *SessionLog) MinutesToday(username string) (float64, error) {
	midnight := time.Now().UTC().Truncate(24 * time.Hour).Format(time.RFC3339Nano)
	var total sql.NullFloat64
	err := sl.st.db.QueryRow(
		`SELECT COALESCE(SUM(minutes), 0) FROM session_log
		 WHERE username = ? AND event_type = ? AND at >= ?`,
		username, audit.TypeDisconnect, midnight).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total.Float64, nil
}
