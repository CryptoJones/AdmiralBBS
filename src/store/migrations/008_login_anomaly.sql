-- Rapid-IP-change ("impossible travel") flagging for SysOp visibility.
--
-- We have no GeoIP database and deliberately make no external calls, so we
-- cannot measure true geographic distance. What we CAN do honestly: remember
-- each user's last login IP + time, and when the next login arrives from a
-- DIFFERENT IP within a short window, record it for the operator to eyeball.
-- This is a visibility aid, never an auto-block (legitimate VPN/roaming users
-- change IPs — see BACKLOG non-goals).
ALTER TABLE user ADD COLUMN last_login_ip TEXT NOT NULL DEFAULT '';

CREATE TABLE login_anomaly (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    prev_ip     TEXT    NOT NULL,
    new_ip      TEXT    NOT NULL,
    gap_seconds INTEGER NOT NULL,
    at          TEXT    NOT NULL
);
CREATE INDEX idx_login_anomaly_at ON login_anomaly(at);
