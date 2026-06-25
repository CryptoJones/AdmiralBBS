-- Resident doors a SysOp installed from a forge RELEASE URL. The BBS downloads
-- the binary matching its own OS/arch, runs it under supervision on a localhost
-- address, and bridges callers to it. Persisted here so they relaunch and
-- re-register on every BBS restart. Operator config (non-secret): plaintext.
CREATE TABLE installed_door (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    name             TEXT NOT NULL UNIQUE,
    source_url       TEXT NOT NULL,            -- the forge releases endpoint the SysOp pasted
    version          TEXT NOT NULL DEFAULT '', -- the installed release tag
    bin_path         TEXT NOT NULL,            -- where the downloaded binary lives
    address          TEXT NOT NULL,            -- localhost bridge address (host:port) the door listens on
    min_access_level INTEGER NOT NULL DEFAULT 0,
    installed_at     TEXT NOT NULL
);
