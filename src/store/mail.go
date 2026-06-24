package store

import (
	"database/sql"
	"errors"
	"time"
)

// PrivateMessage is user-to-user mail. Subject and body are encrypted at rest (🔒).
type PrivateMessage struct {
	ID      int64
	FromID  int64
	ToID    int64
	Subject string
	Body    string
	SentAt  time.Time
	ReadAt  *time.Time
}

// Mail is the private-message repository.
type Mail struct{ st *Store }

// Send delivers a private message.
func (r *Mail) Send(fromID, toID int64, subject, body string) (*PrivateMessage, error) {
	encSubj, err := r.st.seal(subject)
	if err != nil {
		return nil, err
	}
	encBody, err := r.st.seal(body)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	res, err := r.st.db.Exec(
		`INSERT INTO private_message (from_id, to_id, subject, body, sent_at)
		 VALUES (?, ?, ?, ?, ?)`,
		fromID, toID, encSubj, encBody, fmtTime(now))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &PrivateMessage{ID: id, FromID: fromID, ToID: toID, Subject: subject, Body: body, SentAt: now}, nil
}

const pmCols = `id, from_id, to_id, subject, body, sent_at, read_at`

func (r *Mail) scan(row interface{ Scan(...any) error }) (*PrivateMessage, error) {
	var m PrivateMessage
	var encSubj, encBody, sent string
	var read sql.NullString
	if err := row.Scan(&m.ID, &m.FromID, &m.ToID, &encSubj, &encBody, &sent, &read); err != nil {
		return nil, err
	}
	var err error
	if m.Subject, err = r.st.open(encSubj); err != nil {
		return nil, err
	}
	if m.Body, err = r.st.open(encBody); err != nil {
		return nil, err
	}
	m.SentAt = parseTime(sent)
	if read.Valid {
		t := parseTime(read.String)
		m.ReadAt = &t
	}
	return &m, nil
}

// Inbox lists a user's received mail, newest first.
func (r *Mail) Inbox(userID int64) ([]*PrivateMessage, error) {
	rows, err := r.st.db.Query(`SELECT `+pmCols+` FROM private_message WHERE to_id = ? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*PrivateMessage
	for rows.Next() {
		m, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// UnreadCount returns how many unread messages a user has.
func (r *Mail) UnreadCount(userID int64) (int, error) {
	var n int
	err := r.st.db.QueryRow(`SELECT COUNT(*) FROM private_message WHERE to_id = ? AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

// Get fetches a message the user is party to; reading it as the recipient marks
// it read.
func (r *Mail) Get(id, userID int64) (*PrivateMessage, error) {
	row := r.st.db.QueryRow(`SELECT `+pmCols+` FROM private_message WHERE id = ?`, id)
	m, err := r.scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if m.ToID != userID && m.FromID != userID {
		return nil, ErrNotFound // not yours
	}
	if m.ToID == userID && m.ReadAt == nil {
		now := time.Now().UTC()
		if _, err := r.st.db.Exec(`UPDATE private_message SET read_at = ? WHERE id = ?`, fmtTime(now), id); err == nil {
			m.ReadAt = &now
		}
	}
	return m, nil
}

// Delete removes a message from the recipient's mailbox. Only the recipient
// (to_id) may delete it, so a sender can't reach into someone else's inbox.
// Returns ErrNotFound if the id isn't the user's received mail.
func (r *Mail) Delete(id, userID int64) error {
	res, err := r.st.db.Exec(`DELETE FROM private_message WHERE id = ? AND to_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Mail returns the private-message repository.
func (s *Store) Mail() *Mail { return &Mail{st: s} }
