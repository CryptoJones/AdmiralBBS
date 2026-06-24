-- SysOp-customizable BBS settings: a simple key/value store for branding (the
-- BBS name + tagline shown on the main menu) and the message of the day. Values
-- are operator-authored display text (not secret), so they're stored plaintext.
-- Unset keys fall back to code defaults, so no seed rows are required.
CREATE TABLE setting (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);
