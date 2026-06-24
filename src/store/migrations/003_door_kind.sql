-- Door games come in two kinds:
--   'subprocess' (default): the BBS spawns a process per player (single-player,
--                or turn-based file-shared multiplayer via the shared data dir).
--   'resident':  a persistent, real-time multiplayer game server (MajorMUD /
--                Worldgroup style) that runs continuously; the BBS BRIDGES each
--                caller's session to it (net_type+address), so all players share
--                one live world.
ALTER TABLE door ADD COLUMN kind     TEXT NOT NULL DEFAULT 'subprocess';
ALTER TABLE door ADD COLUMN net_type TEXT NOT NULL DEFAULT 'tcp';   -- tcp | unix
ALTER TABLE door ADD COLUMN address  TEXT NOT NULL DEFAULT '';       -- e.g. 127.0.0.1:4000
