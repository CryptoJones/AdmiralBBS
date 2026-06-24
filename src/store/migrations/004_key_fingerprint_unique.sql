-- One account per SSH key: a given public-key fingerprint may be ACTIVE on at
-- most one account at a time. This is the anti-sockpuppet control — a caller
-- can't register the same key on many handles. Enforcement is at the DB layer
-- (partial unique index) so it's race-safe, exactly like the handle UNIQUE
-- constraint; concurrent inserts can't both win.
--
-- Scoped to active keys (revoked_at IS NULL): revoking a key frees its
-- fingerprint, so a user who rotates a key away can later re-add it, and a key
-- that legitimately changes hands isn't blocked forever. Revoked rows are kept
-- for history and are exempt from the constraint.
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_key_active_fingerprint
    ON user_key (fingerprint)
    WHERE revoked_at IS NULL;
