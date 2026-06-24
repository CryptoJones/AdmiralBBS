package store

import (
	"database/sql"
	"strings"
	"time"
)

// Report is a user complaint routed to the SysOp queue.
type Report struct {
	ID         int64
	ReporterID int64
	TargetID   int64
	Context    string // short locator, e.g. "mail #42"
	Note       string
	CreatedAt  time.Time
	ResolvedAt *time.Time
	ResolvedBy sql.NullInt64
}

// Reports is the abuse-report repository.
type Reports struct{ st *Store }

// Reports returns the report repository.
func (s *Store) Reports() *Reports { return &Reports{st: s} }

// File records a report from reporter against target.
func (r *Reports) File(reporterID, targetID int64, context, note string) (*Report, error) {
	res, err := r.st.db.Exec(
		`INSERT INTO report (reporter_id, target_id, context, note, created_at) VALUES (?, ?, ?, ?, ?)`,
		reporterID, targetID, strings.TrimSpace(context), strings.TrimSpace(note),
		fmtTime(time.Now().UTC()))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.byID(id)
}

const reportCols = `id, reporter_id, target_id, context, note, created_at, resolved_at, resolved_by`

func (r *Reports) scan(row interface{ Scan(...any) error }) (*Report, error) {
	var rep Report
	var created string
	var resolved sql.NullString
	if err := row.Scan(&rep.ID, &rep.ReporterID, &rep.TargetID, &rep.Context, &rep.Note,
		&created, &resolved, &rep.ResolvedBy); err != nil {
		return nil, err
	}
	rep.CreatedAt = parseTime(created)
	if resolved.Valid {
		t := parseTime(resolved.String)
		rep.ResolvedAt = &t
	}
	return &rep, nil
}

func (r *Reports) byID(id int64) (*Report, error) {
	return r.scan(r.st.db.QueryRow(`SELECT ` + reportCols + ` FROM report WHERE id = ?`, id))
}

// Open returns unresolved reports, oldest first (FIFO for the SysOp queue).
func (r *Reports) Open() ([]*Report, error) {
	rows, err := r.st.db.Query(`SELECT ` + reportCols + ` FROM report WHERE resolved_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Report
	for rows.Next() {
		rep, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	return out, rows.Err()
}

// OpenCount returns how many reports are awaiting review.
func (r *Reports) OpenCount() (int, error) {
	var n int
	err := r.st.db.QueryRow(`SELECT COUNT(*) FROM report WHERE resolved_at IS NULL`).Scan(&n)
	return n, err
}

// Resolve marks a report handled by a SysOp.
func (r *Reports) Resolve(id, sysopID int64) error {
	_, err := r.st.db.Exec(
		`UPDATE report SET resolved_at = ?, resolved_by = ? WHERE id = ? AND resolved_at IS NULL`,
		fmtTime(time.Now().UTC()), sysopID, id)
	return err
}
