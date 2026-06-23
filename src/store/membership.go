package store

import (
	"database/sql"
	"time"
)

// Membership review decisions.
const (
	DecisionPending  = "pending"
	DecisionApproved = "approved"
	DecisionDenied   = "denied"
)

// Membership is a manual-approval application (see docs/DATA_MODEL.md). The
// exact approval workflow is an open question — this is the storage for it.
type Membership struct {
	ID         int64
	UserID     int64
	AppliedAt  time.Time
	ReviewedBy *int64
	ReviewedAt *time.Time
	Decision   string
	Note       string
}

// Memberships is the membership repository. The free-text note (applicant's
// reason / SysOp remark) is user content and is encrypted at rest.
type Memberships struct{ st *Store }

// Apply records a new pending membership application with the applicant's note.
func (r *Memberships) Apply(userID int64, note string) (*Membership, error) {
	encNote, err := r.st.seal(note)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	res, err := r.st.db.Exec(
		`INSERT INTO membership (user_id, applied_at, decision, note) VALUES (?, ?, ?, ?)`,
		userID, fmtTime(now), DecisionPending, encNote)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.ByID(id)
}

const membershipCols = `id, user_id, applied_at, reviewed_by, reviewed_at, decision, note`

func (r *Memberships) scan(row interface{ Scan(...any) error }) (*Membership, error) {
	var m Membership
	var applied string
	var reviewedBy sql.NullInt64
	var reviewedAt sql.NullString
	var encNote string
	if err := row.Scan(&m.ID, &m.UserID, &applied, &reviewedBy, &reviewedAt, &m.Decision, &encNote); err != nil {
		return nil, err
	}
	note, err := r.st.open(encNote)
	if err != nil {
		return nil, err
	}
	m.Note = note
	m.AppliedAt = parseTime(applied)
	if reviewedBy.Valid {
		m.ReviewedBy = &reviewedBy.Int64
	}
	if reviewedAt.Valid {
		t := parseTime(reviewedAt.String)
		m.ReviewedAt = &t
	}
	return &m, nil
}

// ByID looks up a membership application.
func (r *Memberships) ByID(id int64) (*Membership, error) {
	row := r.st.db.QueryRow(`SELECT `+membershipCols+` FROM membership WHERE id = ?`, id)
	return r.scan(row)
}

// Review records a SysOp decision (approved/denied) on an application.
func (r *Memberships) Review(id, reviewerID int64, decision, note string) error {
	encNote, err := r.st.seal(note)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	_, err = r.st.db.Exec(
		`UPDATE membership SET reviewed_by = ?, reviewed_at = ?, decision = ?, note = ? WHERE id = ?`,
		reviewerID, fmtTime(now), decision, encNote, id)
	return err
}

// Pending returns all pending applications, oldest first.
func (r *Memberships) Pending() ([]*Membership, error) {
	rows, err := r.st.db.Query(`SELECT `+membershipCols+` FROM membership WHERE decision = ? ORDER BY id`, DecisionPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Membership
	for rows.Next() {
		m, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
