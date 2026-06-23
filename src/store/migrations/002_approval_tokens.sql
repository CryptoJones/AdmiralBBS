-- One-time approval tokens (SEC-2). Issued when a SysOp approves an applicant;
-- relayed out-of-band; redeemed once, on the applicant's first SSH login, to set
-- their password. Stored HASHED (sha256 of a 256-bit random token) — never
-- plaintext. Single-use (used_at) and time-limited (expires_at).
CREATE TABLE approval_token (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    token_hash TEXT    NOT NULL,
    expires_at TEXT    NOT NULL,
    used_at    TEXT
);
CREATE INDEX idx_approval_token_user ON approval_token(user_id);
