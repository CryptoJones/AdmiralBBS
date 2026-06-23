package tests

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"admiralbbs/src/store"
)

func TestTokenIssueRedeemSingleUse(t *testing.T) {
	s, _ := openTestStore(t)
	u, _ := s.Users().Create("alice", "", "", "")
	tokens := s.Tokens()

	plain, err := tokens.Issue(u.ID)
	if err != nil || plain == "" {
		t.Fatalf("issue: %v plain=%q", err, plain)
	}

	// Wrong token rejected.
	if err := tokens.Redeem(u.ID, "not-the-token"); err != store.ErrTokenInvalid {
		t.Fatalf("wrong token: want ErrTokenInvalid, got %v", err)
	}
	// Correct token accepted once.
	if err := tokens.Redeem(u.ID, plain); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	// Single-use: second redeem fails.
	if err := tokens.Redeem(u.ID, plain); err != store.ErrTokenInvalid {
		t.Fatalf("reused token: want ErrTokenInvalid, got %v", err)
	}
}

func TestTokenExpiry(t *testing.T) {
	s, _ := openTestStore(t)
	u, _ := s.Users().Create("bob", "", "", "")

	// Insert an already-expired token (hash matches store.hashToken's scheme).
	sum := sha256.Sum256([]byte("expired-token"))
	hash := base64.RawStdEncoding.EncodeToString(sum[:])
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano)
	if _, err := s.DB().Exec(
		`INSERT INTO approval_token (user_id, token_hash, expires_at) VALUES (?, ?, ?)`,
		u.ID, hash, past); err != nil {
		t.Fatal(err)
	}
	if err := s.Tokens().Redeem(u.ID, "expired-token"); err != store.ErrTokenInvalid {
		t.Fatalf("expired token: want ErrTokenInvalid, got %v", err)
	}
}
