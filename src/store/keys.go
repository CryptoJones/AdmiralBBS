package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ErrKeyTaken means the offered public key is already active on another account
// in the SAME tier. A key may map to at most one SysOp-tier account and one
// regular account (anti-sockpuppet, but operator-friendly).
var ErrKeyTaken = errors.New("store: that SSH key is already registered to another account at this access tier")

// Key is one registered SSH public key for a user. Public keys are not secret,
// so they are stored in normalised authorized_keys form (cleartext); the
// encrypted volume covers them at rest.
type Key struct {
	ID          int64
	UserID      int64
	PublicKey   string // normalised "ssh-ed25519 AAAA... comment"
	Fingerprint string // SHA256:...
	Comment     string
	AddedAt     time.Time
	RevokedAt   *time.Time
}

// Keys is the SSH-key repository. Users may register many keys and add/revoke
// them over time (revocation is soft).
type Keys struct{ st *Store }

// ValidatePublicKey reports whether line parses as an SSH authorized_keys entry.
func ValidatePublicKey(line string) error {
	_, _, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
	return err
}

// Add parses, normalises, and stores an authorized_keys line for the user.
func (r *Keys) Add(userID int64, authorizedKey string) (*Key, error) {
	pub, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(authorizedKey))
	if err != nil {
		return nil, fmt.Errorf("invalid SSH public key: %w", err)
	}
	normalised := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub)))
	fp := ssh.FingerprintSHA256(pub)
	now := time.Now().UTC()

	// One account per TIER per key: a fingerprint may be active on at most one
	// SysOp-tier account and one regular account. Serialize so the check+insert
	// is atomic against a concurrent registration of the same key.
	r.st.keyMu.Lock()
	defer r.st.keyMu.Unlock()

	var targetLevel int
	if err := r.st.db.QueryRow(`SELECT access_level FROM user WHERE id = ?`, userID).Scan(&targetLevel); err != nil {
		return nil, err
	}
	targetSysop := targetLevel >= SysOpLevel
	rows, err := r.st.db.Query(
		`SELECT u.access_level FROM user_key k JOIN user u ON u.id = k.user_id
		 WHERE k.fingerprint = ? AND k.revoked_at IS NULL AND k.user_id != ?`, fp, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var lvl int
		if err := rows.Scan(&lvl); err != nil {
			return nil, err
		}
		if (lvl >= SysOpLevel) == targetSysop { // same tier already holds this key
			return nil, ErrKeyTaken
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	res, err := r.st.db.Exec(
		`INSERT INTO user_key (user_id, public_key, fingerprint, comment, added_at)
		 VALUES (?, ?, ?, ?, ?)`,
		userID, normalised, fp, comment, fmtTime(now))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return r.byID(id)
}

const keyCols = `id, user_id, public_key, fingerprint, comment, added_at, revoked_at`

func (r *Keys) scan(row interface{ Scan(...any) error }) (*Key, error) {
	var k Key
	var added string
	var revoked sql.NullString
	if err := row.Scan(&k.ID, &k.UserID, &k.PublicKey, &k.Fingerprint, &k.Comment, &added, &revoked); err != nil {
		return nil, err
	}
	k.AddedAt = parseTime(added)
	if revoked.Valid {
		t := parseTime(revoked.String)
		k.RevokedAt = &t
	}
	return &k, nil
}

func (r *Keys) byID(id int64) (*Key, error) {
	row := r.st.db.QueryRow(`SELECT `+keyCols+` FROM user_key WHERE id = ?`, id)
	return r.scan(row)
}

// ByFingerprint returns the active key with the given SHA256 fingerprint, or
// (nil, nil) if no account currently holds it. Because a fingerprint is unique
// among active keys (migration 004), this resolves a key to its single owner.
func (r *Keys) ByFingerprint(fp string) (*Key, error) {
	row := r.st.db.QueryRow(
		`SELECT `+keyCols+` FROM user_key WHERE fingerprint = ? AND revoked_at IS NULL`, fp)
	k, err := r.scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return k, err
}

// Active returns the user's non-revoked keys.
func (r *Keys) Active(userID int64) ([]*Key, error) {
	return r.list(`SELECT `+keyCols+` FROM user_key WHERE user_id = ? AND revoked_at IS NULL ORDER BY id`, userID)
}

// All returns every key for the user, including revoked ones (for SysOp view).
func (r *Keys) All(userID int64) ([]*Key, error) {
	return r.list(`SELECT `+keyCols+` FROM user_key WHERE user_id = ? ORDER BY id`, userID)
}

func (r *Keys) list(query string, args ...any) ([]*Key, error) {
	rows, err := r.st.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Key
	for rows.Next() {
		k, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// Revoke soft-deletes a key (sets revoked_at; the row is kept for history).
func (r *Keys) Revoke(id int64) error {
	_, err := r.st.db.Exec(`UPDATE user_key SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`,
		fmtTime(time.Now().UTC()), id)
	return err
}

// Authorizes reports whether the offered public key matches one of the user's
// ACTIVE keys — the SSH-layer half of two-factor auth (used in Sprint 003).
func (r *Keys) Authorizes(userID int64, offered ssh.PublicKey) (bool, error) {
	want := ssh.FingerprintSHA256(offered)
	active, err := r.Active(userID)
	if err != nil {
		return false, err
	}
	for _, k := range active {
		if k.Fingerprint == want {
			return true, nil
		}
	}
	return false, nil
}
