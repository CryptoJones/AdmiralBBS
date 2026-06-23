package store

import (
	"database/sql"
	"errors"
	"time"
)

// MessageArea is a message board ("base"). Tables already exist from migration 001.
type MessageArea struct {
	ID             int64
	Name           string
	Description    string
	MinAccessLevel int
}

// Message is a post or a threaded reply. Subject and body are user content and
// are encrypted at rest (🔒).
type Message struct {
	ID       int64
	AreaID   int64
	AuthorID int64
	ParentID *int64
	Subject  string
	Body     string
	PostedAt time.Time
}

// MessageAreas is the message-board repository.
type MessageAreas struct{ st *Store }

// Create adds a board.
func (r *MessageAreas) Create(name, description string, minLevel int) (*MessageArea, error) {
	res, err := r.st.db.Exec(
		`INSERT INTO message_area (name, description, min_access_level) VALUES (?, ?, ?)`,
		name, description, minLevel)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &MessageArea{ID: id, Name: name, Description: description, MinAccessLevel: minLevel}, nil
}

// Visible lists areas the given access level may enter, by name.
func (r *MessageAreas) Visible(accessLevel int) ([]*MessageArea, error) {
	rows, err := r.st.db.Query(
		`SELECT id, name, description, min_access_level FROM message_area
		 WHERE min_access_level <= ? ORDER BY name`, accessLevel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*MessageArea
	for rows.Next() {
		var a MessageArea
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.MinAccessLevel); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

// Count returns the number of areas (used to decide whether to seed defaults).
func (r *MessageAreas) Count() (int, error) {
	var n int
	err := r.st.db.QueryRow(`SELECT COUNT(*) FROM message_area`).Scan(&n)
	return n, err
}

// ByID fetches one area (enforces the caller's access level).
func (r *MessageAreas) ByID(id int64, accessLevel int) (*MessageArea, error) {
	var a MessageArea
	err := r.st.db.QueryRow(
		`SELECT id, name, description, min_access_level FROM message_area WHERE id = ?`, id).
		Scan(&a.ID, &a.Name, &a.Description, &a.MinAccessLevel)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if accessLevel < a.MinAccessLevel {
		return nil, ErrNotFound // hide what they can't reach
	}
	return &a, nil
}

// Messages is the message repository.
type Messages struct{ st *Store }

// Post stores a new message (or reply when parentID != nil). Subject and body
// are sealed at rest.
func (r *Messages) Post(areaID, authorID int64, parentID *int64, subject, body string) (*Message, error) {
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
		`INSERT INTO message (area_id, author_id, parent_id, subject, body, posted_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		areaID, authorID, parentID, encSubj, encBody, fmtTime(now))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Message{ID: id, AreaID: areaID, AuthorID: authorID, ParentID: parentID, Subject: subject, Body: body, PostedAt: now}, nil
}

const messageCols = `id, area_id, author_id, parent_id, subject, body, posted_at`

func (r *Messages) scan(row interface{ Scan(...any) error }) (*Message, error) {
	var m Message
	var parent sql.NullInt64
	var encSubj, encBody, posted string
	if err := row.Scan(&m.ID, &m.AreaID, &m.AuthorID, &parent, &encSubj, &encBody, &posted); err != nil {
		return nil, err
	}
	var err error
	if m.Subject, err = r.st.open(encSubj); err != nil {
		return nil, err
	}
	if m.Body, err = r.st.open(encBody); err != nil {
		return nil, err
	}
	if parent.Valid {
		m.ParentID = &parent.Int64
	}
	m.PostedAt = parseTime(posted)
	return &m, nil
}

// Thread lists top-level messages in an area (oldest first).
func (r *Messages) Thread(areaID int64) ([]*Message, error) {
	return r.list(`SELECT `+messageCols+` FROM message WHERE area_id = ? AND parent_id IS NULL ORDER BY id`, areaID)
}

// Replies lists replies to a parent message (oldest first).
func (r *Messages) Replies(parentID int64) ([]*Message, error) {
	return r.list(`SELECT `+messageCols+` FROM message WHERE parent_id = ? ORDER BY id`, parentID)
}

// ByID fetches one message.
func (r *Messages) ByID(id int64) (*Message, error) {
	row := r.st.db.QueryRow(`SELECT `+messageCols+` FROM message WHERE id = ?`, id)
	m, err := r.scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return m, err
}

func (r *Messages) list(query string, args ...any) ([]*Message, error) {
	rows, err := r.st.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Message
	for rows.Next() {
		m, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// EnsureSeedAreas creates a couple of default boards on first run so a fresh BBS
// has somewhere to post. SysOps can add more (Sprint 008).
func (s *Store) EnsureSeedAreas() error {
	n, err := s.MessageAreas().Count()
	if err != nil || n > 0 {
		return err
	}
	for _, a := range []struct {
		name, desc string
	}{
		{"General", "General chatter and introductions"},
		{"Retro Computing", "BBSes, ANSI art, old iron"},
	} {
		if _, err := s.MessageAreas().Create(a.name, a.desc, 0); err != nil {
			return err
		}
	}
	return nil
}

// MessageAreas returns the board repository.
func (s *Store) MessageAreas() *MessageAreas { return &MessageAreas{st: s} }

// Messages returns the message repository.
func (s *Store) Messages() *Messages { return &Messages{st: s} }
