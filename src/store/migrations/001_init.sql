-- AdmiralBBS initial schema (migration 1).
-- Timestamps are stored as RFC3339 TEXT so they match the JSONL audit trail
-- and read cleanly. Integer primary keys (BBS user-number culture).

CREATE TABLE user (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    handle        TEXT    NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT    NOT NULL,
    real_name     TEXT    NOT NULL DEFAULT '',
    email         TEXT    NOT NULL DEFAULT '',
    access_level  INTEGER NOT NULL DEFAULT 0,
    status        TEXT    NOT NULL DEFAULT 'pending',
    daily_minutes INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT    NOT NULL,
    last_login_at TEXT
);

-- A user may register MANY SSH public keys; keys are added/revoked over time.
-- Revocation is soft (revoked_at set) so the history is preserved. Public keys
-- are not secret, so they are stored in authorized_keys form (cleartext);
-- the encrypted volume covers them at rest. SSH auth (Sprint 003) checks an
-- offered key against the user's ACTIVE (revoked_at IS NULL) keys, and the BBS
-- still requires the password too (two-factor).
CREATE TABLE user_key (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    public_key  TEXT    NOT NULL,            -- normalised authorized_keys line
    fingerprint TEXT    NOT NULL,            -- SHA256 fingerprint (display/dedup)
    comment     TEXT    NOT NULL DEFAULT '',
    added_at    TEXT    NOT NULL,
    revoked_at  TEXT                          -- NULL = active
);
CREATE INDEX idx_user_key_user ON user_key(user_id);
CREATE INDEX idx_user_key_fp   ON user_key(fingerprint);

CREATE TABLE membership (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    applied_at  TEXT    NOT NULL,
    reviewed_by INTEGER REFERENCES user(id),
    reviewed_at TEXT,
    decision    TEXT    NOT NULL DEFAULT 'pending',
    note        TEXT    NOT NULL DEFAULT ''
);
CREATE INDEX idx_membership_user ON membership(user_id);

CREATE TABLE message_area (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    name             TEXT    NOT NULL UNIQUE,
    description      TEXT    NOT NULL DEFAULT '',
    min_access_level INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE message (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    area_id   INTEGER NOT NULL REFERENCES message_area(id) ON DELETE CASCADE,
    author_id INTEGER NOT NULL REFERENCES user(id),
    parent_id INTEGER REFERENCES message(id),
    subject   TEXT    NOT NULL DEFAULT '',
    body      TEXT    NOT NULL DEFAULT '',
    posted_at TEXT    NOT NULL
);
CREATE INDEX idx_message_area   ON message(area_id);
CREATE INDEX idx_message_parent ON message(parent_id);

CREATE TABLE private_message (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    from_id INTEGER NOT NULL REFERENCES user(id),
    to_id   INTEGER NOT NULL REFERENCES user(id),
    subject TEXT    NOT NULL DEFAULT '',
    body    TEXT    NOT NULL DEFAULT '',
    sent_at TEXT    NOT NULL,
    read_at TEXT
);
CREATE INDEX idx_pm_to ON private_message(to_id);

CREATE TABLE file_area (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    name             TEXT    NOT NULL UNIQUE,
    min_access_level INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE file_entry (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    area_id        INTEGER NOT NULL REFERENCES file_area(id) ON DELETE CASCADE,
    filename       TEXT    NOT NULL,
    path           TEXT    NOT NULL,
    size_bytes     INTEGER NOT NULL DEFAULT 0,
    description    TEXT    NOT NULL DEFAULT '',
    uploader_id    INTEGER REFERENCES user(id),
    download_count INTEGER NOT NULL DEFAULT 0,
    uploaded_at    TEXT    NOT NULL
);
CREATE INDEX idx_file_area ON file_entry(area_id);

CREATE TABLE door (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    name             TEXT    NOT NULL UNIQUE,
    command          TEXT    NOT NULL,
    dropfile_format  TEXT    NOT NULL DEFAULT 'door32.sys',
    min_access_level INTEGER NOT NULL DEFAULT 0
);

-- session_log MIRRORS the authoritative append-only JSONL audit trail: one row
-- per event (connect | activity | disconnect). The JSONL file is the source of
-- truth; this table exists for SysOp queryability and redundancy.
CREATE TABLE session_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT    NOT NULL,
    user_id    INTEGER REFERENCES user(id),
    username   TEXT    NOT NULL DEFAULT '',
    transport  TEXT    NOT NULL,
    remote_ip  TEXT    NOT NULL DEFAULT '',
    event_type TEXT    NOT NULL,
    action     TEXT    NOT NULL DEFAULT '',
    detail     TEXT    NOT NULL DEFAULT '',
    minutes    REAL    NOT NULL DEFAULT 0,
    at         TEXT    NOT NULL
);
CREATE INDEX idx_session_log_session ON session_log(session_id);
CREATE INDEX idx_session_log_user    ON session_log(user_id);
