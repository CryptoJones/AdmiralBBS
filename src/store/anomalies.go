package store

import "time"

// LoginAnomaly is a recorded rapid-IP-change ("impossible travel") event.
type LoginAnomaly struct {
	ID         int64
	UserID     int64
	PrevIP     string
	NewIP      string
	GapSeconds int64
	At         time.Time
}

// Anomalies is the login-anomaly repository (SysOp visibility).
type Anomalies struct{ st *Store }

// Anomalies returns the login-anomaly repository.
func (s *Store) Anomalies() *Anomalies { return &Anomalies{st: s} }

// Recent returns the most recent anomalies, newest first.
func (r *Anomalies) Recent(limit int) ([]*LoginAnomaly, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.st.db.Query(
		`SELECT id, user_id, prev_ip, new_ip, gap_seconds, at
		 FROM login_anomaly ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*LoginAnomaly
	for rows.Next() {
		var a LoginAnomaly
		var at string
		if err := rows.Scan(&a.ID, &a.UserID, &a.PrevIP, &a.NewIP, &a.GapSeconds, &at); err != nil {
			return nil, err
		}
		a.At = parseTime(at)
		out = append(out, &a)
	}
	return out, rows.Err()
}

// Count returns the total number of recorded anomalies.
func (r *Anomalies) Count() (int, error) {
	var n int
	err := r.st.db.QueryRow(`SELECT COUNT(*) FROM login_anomaly`).Scan(&n)
	return n, err
}
