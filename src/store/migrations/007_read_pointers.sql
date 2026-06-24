-- Per-user "new since last visit" markers for message boards. One row per
-- (user, area) records the highest message id that user has seen in that area.
-- A message with id > last_seen_id is NEW to them. Classic BBS behaviour.
CREATE TABLE board_read (
    user_id      INTEGER NOT NULL REFERENCES user(id) ON DELETE CASCADE,
    area_id      INTEGER NOT NULL REFERENCES message_area(id) ON DELETE CASCADE,
    last_seen_id INTEGER NOT NULL DEFAULT 0,
    updated_at   TEXT    NOT NULL,
    PRIMARY KEY (user_id, area_id)
);
