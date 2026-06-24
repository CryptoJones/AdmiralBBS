package store

import (
	"database/sql"
	"errors"
	"time"
)

// ReadPointers tracks, per user and board, the highest message id seen — the
// basis for "N new since last visit".
type ReadPointers struct{ st *Store }

// ReadPointers returns the read-pointer repository.
func (s *Store) ReadPointers() *ReadPointers { return &ReadPointers{st: s} }

// LastSeen returns the highest message id the user has seen in the area (0 if
// they've never visited).
func (r *ReadPointers) LastSeen(userID, areaID int64) (int64, error) {
	var id int64
	err := r.st.db.QueryRow(
		`SELECT last_seen_id FROM board_read WHERE user_id = ? AND area_id = ?`,
		userID, areaID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

// Mark records that the user has now seen up to maxID in the area. It never
// moves the pointer backwards (a stale lower value can't un-read newer posts).
func (r *ReadPointers) Mark(userID, areaID, maxID int64) error {
	_, err := r.st.db.Exec(
		`INSERT INTO board_read (user_id, area_id, last_seen_id, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id, area_id) DO UPDATE SET
		   last_seen_id = MAX(last_seen_id, excluded.last_seen_id),
		   updated_at   = excluded.updated_at`,
		userID, areaID, maxID, fmtTime(time.Now().UTC()))
	return err
}

// NewCount returns how many messages in the area are newer than the user's
// read pointer.
func (r *ReadPointers) NewCount(userID, areaID int64) (int, error) {
	last, err := r.LastSeen(userID, areaID)
	if err != nil {
		return 0, err
	}
	var n int
	err = r.st.db.QueryRow(
		`SELECT COUNT(*) FROM message WHERE area_id = ? AND id > ?`, areaID, last).Scan(&n)
	return n, err
}

// MaxMessageID returns the highest message id in an area (0 if empty) — the
// value to Mark when a user has browsed the area.
func (r *ReadPointers) MaxMessageID(areaID int64) (int64, error) {
	var id int64
	err := r.st.db.QueryRow(
		`SELECT COALESCE(MAX(id), 0) FROM message WHERE area_id = ?`, areaID).Scan(&id)
	return id, err
}
