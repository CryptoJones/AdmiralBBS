-- Caller "points" — a running score shown on a member's stats/profile screen.
-- SysOps award (or dock) points from the user-management panel; the value is a
-- plain integer (not PII), so it lives unencrypted alongside the other account
-- columns. Defaults to 0 so existing accounts start with a clean slate.
ALTER TABLE user ADD COLUMN points INTEGER NOT NULL DEFAULT 0;
