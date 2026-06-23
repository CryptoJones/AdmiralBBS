package store

import (
	"database/sql"
	"errors"
	"time"
)

// User account statuses.
const (
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusDenied    = "denied"
	StatusSuspended = "suspended"
)

// ErrNotFound is returned when a lookup matches no row.
var ErrNotFound = errors.New("not found")

// User is a caller account (see docs/DATA_MODEL.md).
type User struct {
	ID           int64
	Handle       string
	PasswordHash string
	RealName     string
	Email        string
	AccessLevel  int
	Status       string
	DailyMinutes int
	CreatedAt    time.Time
	LastLoginAt  *time.Time
}

// Users is the user repository. real_name and email are PII and are encrypted
// at rest via the store's vault (RISKS: app-level AEAD layer).
type Users struct{ st *Store }

// Create inserts a new pending user. The password must already be hashed
// (use HashPassword) and may be empty for a Telnet applicant who will set it on
// first SSH login. Returns the created user with its assigned id.
func (r *Users) Create(handle, passwordHash, realName, email string) (*User, error) {
	encName, err := r.st.seal(realName)
	if err != nil {
		return nil, err
	}
	encEmail, err := r.st.seal(email)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	res, err := r.st.db.Exec(
		`INSERT INTO user (handle, password_hash, real_name, email, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		handle, passwordHash, encName, encEmail, StatusPending, fmtTime(now))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.ByID(id)
}

const userCols = `id, handle, password_hash, real_name, email, access_level, status, daily_minutes, created_at, last_login_at`

func (r *Users) scan(row interface{ Scan(...any) error }) (*User, error) {
	var u User
	var created string
	var lastLogin sql.NullString
	var encName, encEmail string
	if err := row.Scan(&u.ID, &u.Handle, &u.PasswordHash, &encName, &encEmail,
		&u.AccessLevel, &u.Status, &u.DailyMinutes, &created, &lastLogin); err != nil {
		return nil, err
	}
	var err error
	if u.RealName, err = r.st.open(encName); err != nil {
		return nil, err
	}
	if u.Email, err = r.st.open(encEmail); err != nil {
		return nil, err
	}
	u.CreatedAt = parseTime(created)
	if lastLogin.Valid {
		t := parseTime(lastLogin.String)
		u.LastLoginAt = &t
	}
	return &u, nil
}

// ByID looks up a user by id.
func (r *Users) ByID(id int64) (*User, error) {
	row := r.st.db.QueryRow(`SELECT `+userCols+` FROM user WHERE id = ?`, id)
	u, err := r.scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

// ByHandle looks up a user by handle (case-insensitive).
func (r *Users) ByHandle(handle string) (*User, error) {
	row := r.st.db.QueryRow(`SELECT `+userCols+` FROM user WHERE handle = ? COLLATE NOCASE`, handle)
	u, err := r.scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

// SetStatus updates a user's status and access level (used by SysOp approval).
func (r *Users) SetStatus(id int64, status string, accessLevel int) error {
	_, err := r.st.db.Exec(`UPDATE user SET status = ?, access_level = ? WHERE id = ?`,
		status, accessLevel, id)
	return err
}

// SetPassword stores a new password hash (first-SSH-login onboarding, SEC-2).
func (r *Users) SetPassword(id int64, passwordHash string) error {
	_, err := r.st.db.Exec(`UPDATE user SET password_hash = ? WHERE id = ?`, passwordHash, id)
	return err
}

// TouchLogin records the user's most recent login time.
func (r *Users) TouchLogin(id int64, when time.Time) error {
	_, err := r.st.db.Exec(`UPDATE user SET last_login_at = ? WHERE id = ?`,
		fmtTime(when.UTC()), id)
	return err
}

// ListByStatus returns users in a given status, oldest first.
func (r *Users) ListByStatus(status string) ([]*User, error) {
	rows, err := r.st.db.Query(`SELECT `+userCols+` FROM user WHERE status = ? ORDER BY id`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		u, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func fmtTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
