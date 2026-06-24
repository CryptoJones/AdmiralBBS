-- SysOp IP banlist. A banned source is rejected by BOTH transports (Telnet and
-- SSH) at accept time, before any authentication — the cheapest place to shed an
-- abusive caller. A pattern is either an exact IP ("203.0.113.7") or a CIDR
-- block ("203.0.113.0/24"); matching is done in Go (store.Bans.IsBanned).
-- Bans are soft-liftable: lifted_at is set rather than the row deleted, so the
-- history of who-banned-what survives for the audit story.
CREATE TABLE ip_ban (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    pattern    TEXT    NOT NULL,            -- exact IP or CIDR
    reason     TEXT    NOT NULL DEFAULT '',
    banned_by  INTEGER REFERENCES user(id),
    banned_at  TEXT    NOT NULL,
    lifted_at  TEXT                          -- NULL = active
);
CREATE INDEX idx_ip_ban_active ON ip_ban(lifted_at);
