package store

import "time"

// Blocks is the personal-mute repository. A block is one-directional and private
// to the blocker: they stop seeing the blocked user's mail and board posts.
type Blocks struct{ st *Store }

// Blocks returns the block repository.
func (s *Store) Blocks() *Blocks { return &Blocks{st: s} }

// Block mutes blocked for blocker. Idempotent (re-blocking is a no-op).
func (r *Blocks) Block(blockerID, blockedID int64) error {
	if blockerID == blockedID {
		return nil // can't block yourself
	}
	_, err := r.st.db.Exec(
		`INSERT OR IGNORE INTO user_block (blocker_id, blocked_id, created_at) VALUES (?, ?, ?)`,
		blockerID, blockedID, fmtTime(time.Now().UTC()))
	return err
}

// Unblock removes a mute. No-op if it wasn't set.
func (r *Blocks) Unblock(blockerID, blockedID int64) error {
	_, err := r.st.db.Exec(
		`DELETE FROM user_block WHERE blocker_id = ? AND blocked_id = ?`, blockerID, blockedID)
	return err
}

// IsBlocked reports whether blocker has muted blocked.
func (r *Blocks) IsBlocked(blockerID, blockedID int64) (bool, error) {
	var n int
	err := r.st.db.QueryRow(
		`SELECT COUNT(*) FROM user_block WHERE blocker_id = ? AND blocked_id = ?`,
		blockerID, blockedID).Scan(&n)
	return n > 0, err
}

// BlockedSet returns the set of user ids the blocker has muted — handy for
// filtering lists of mail/posts in one pass.
func (r *Blocks) BlockedSet(blockerID int64) (map[int64]bool, error) {
	rows, err := r.st.db.Query(`SELECT blocked_id FROM user_block WHERE blocker_id = ?`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	set := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		set[id] = true
	}
	return set, rows.Err()
}

// List returns the user ids the blocker has muted, oldest first.
func (r *Blocks) List(blockerID int64) ([]int64, error) {
	rows, err := r.st.db.Query(
		`SELECT blocked_id FROM user_block WHERE blocker_id = ? ORDER BY created_at`, blockerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
