-- User-to-user moderation.
--
-- user_block: a personal mute. blocker no longer sees blocked's private mail or
-- board posts. It is one-directional and private to the blocker (the blocked
-- user is not notified). UNIQUE keeps it idempotent.
CREATE TABLE user_block (
    blocker_id INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    blocked_id INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    created_at TEXT    NOT NULL,
    PRIMARY KEY (blocker_id, blocked_id)
);

-- report: a complaint routed to the SysOp queue. target_id is the reported
-- user; context is a short locator (e.g. "mail #42" / "board post #7") and note
-- is the reporter's description. Resolution is recorded, not deleted.
CREATE TABLE report (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    reporter_id INTEGER NOT NULL REFERENCES user(id),
    target_id   INTEGER NOT NULL REFERENCES user(id),
    context     TEXT    NOT NULL DEFAULT '',
    note        TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL,
    resolved_at TEXT,                            -- NULL = open
    resolved_by INTEGER REFERENCES user(id)
);
CREATE INDEX idx_report_open ON report(resolved_at);
