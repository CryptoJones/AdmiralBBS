package store

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"time"
)

// TokenTTL is how long a one-time approval token stays valid.
const TokenTTL = 72 * time.Hour

// ErrTokenInvalid is returned for a missing, wrong, expired, or already-used token.
var ErrTokenInvalid = errors.New("invalid or expired token")

// Tokens is the one-time approval-token repository (SEC-2). Tokens are stored
// hashed; the plaintext is returned once (for the SysOp to relay out-of-band).
type Tokens struct{ st *Store }

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}

// Issue creates a fresh single-use token for the user and returns the plaintext
// exactly once. Only the hash is persisted.
func (r *Tokens) Issue(userID int64) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	plain := base64.RawURLEncoding.EncodeToString(raw)
	exp := time.Now().UTC().Add(TokenTTL)
	if _, err := r.st.db.Exec(
		`INSERT INTO approval_token (user_id, token_hash, expires_at) VALUES (?, ?, ?)`,
		userID, hashToken(plain), fmtTime(exp)); err != nil {
		return "", err
	}
	return plain, nil
}

// Redeem consumes a valid, unexpired, unused token for the user. It returns
// ErrTokenInvalid otherwise. On success the token is marked used.
func (r *Tokens) Redeem(userID int64, plain string) error {
	want := hashToken(plain)
	rows, err := r.st.db.Query(
		`SELECT id, token_hash, expires_at FROM approval_token
		 WHERE user_id = ? AND used_at IS NULL`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	now := time.Now().UTC()
	var matchID int64 = -1
	for rows.Next() {
		var id int64
		var hash, exp string
		if err := rows.Scan(&id, &hash, &exp); err != nil {
			return err
		}
		if subtle.ConstantTimeCompare([]byte(hash), []byte(want)) == 1 && now.Before(parseTime(exp)) {
			matchID = id
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if matchID < 0 {
		return ErrTokenInvalid
	}

	res, err := r.st.db.Exec(
		`UPDATE approval_token SET used_at = ? WHERE id = ? AND used_at IS NULL`,
		fmtTime(now), matchID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrTokenInvalid // raced — already used
	}
	return nil
}
