# Data Model — AdmiralBBS

> Embedded SQLite (pure-Go `modernc.org/sqlite`, WAL). Repository interfaces in
> `src/store/` wrap these so subsystems never write raw SQL. Timestamps are
> RFC3339 TEXT; integer PKs (BBS user-number culture).
>
> **Encryption at rest:** fields marked 🔒 are sealed with the `crypto.Vault`
> (XChaCha20-Poly1305) before storage — they are ciphertext on disk. Structural
> columns (ids, handles, timestamps) stay cleartext so the DB can index them;
> the encrypted volume covers those offline. Passwords are argon2id hashes (not
> reversible, not "encryption"). See `planning/DECISIONS.md`.

## Entities

### user
The caller account. Membership status gates access (see `membership`).

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| handle | TEXT UNIQUE | the BBS alias |
| password_hash | TEXT | argon2id — **never** plaintext; empty until set on first SSH login |
| real_name | TEXT 🔒 | optional PII, sealed at rest |
| email | TEXT 🔒 | optional PII, sealed at rest |
| access_level | INTEGER | SysOp / Co-SysOp / Member / Guest (see `docs/PERMISSIONS.md`) |
| status | TEXT | `pending` \| `approved` \| `denied` \| `suspended` |
| daily_minutes | INTEGER | per-day time budget (open question — default TBD) |
| created_at | TEXT | |
| last_login_at | TEXT | |

### user_key
A user's registered SSH public keys (one user → many). Two-factor SSH auth
checks an offered key against the user's ACTIVE keys; revocation is soft.
Public keys are not secret, so they are stored cleartext (the volume covers
them offline).

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| user_id | FK → user | |
| public_key | TEXT | normalised authorized_keys line |
| fingerprint | TEXT | SHA256 fingerprint (display/dedup/match) |
| comment | TEXT | from the key line |
| added_at | TEXT | |
| revoked_at | TEXT | NULL = active |

### membership
The manual-approval workflow for new applicants.

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| user_id | FK → user | |
| applied_at | TEXT | |
| reviewed_by | FK → user | the SysOp who acted |
| reviewed_at | TEXT | |
| decision | TEXT | `pending` \| `approved` \| `denied` |
| note | TEXT 🔒 | applicant reason / SysOp remark — user content, sealed at rest |

### message_area
A message board ("base"). Many messages belong to one area.

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| name | TEXT | e.g. "General", "Retro Computing" |
| description | TEXT | |
| min_access_level | INTEGER | who can read/post |

### message
A post in a board, or a reply (threaded via `parent_id`).

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| area_id | FK → message_area | |
| author_id | FK → user | |
| parent_id | FK → message | NULL = top of thread |
| subject | TEXT | |
| body | TEXT | |
| posted_at | TIMESTAMP | |

### private_message
User-to-user mail.

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| from_id | FK → user | |
| to_id | FK → user | |
| subject | TEXT | |
| body | TEXT | |
| sent_at | TIMESTAMP | |
| read_at | TIMESTAMP | NULL = unread |

### file_area
A download area in the file library.

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| name | TEXT | |
| min_access_level | INTEGER | |

### file_entry
A downloadable object.

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| area_id | FK → file_area | |
| filename | TEXT | |
| path | TEXT | on-disk location (outside DB) |
| size_bytes | INTEGER | |
| description | TEXT | |
| uploader_id | FK → user | |
| download_count | INTEGER | |
| uploaded_at | TIMESTAMP | |

### door
A registered door game.

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| name | TEXT | |
| command | TEXT | executable to launch (sandboxed) |
| dropfile_format | TEXT | `door32.sys` \| `DOOR.SYS` \| ... |
| min_access_level | INTEGER | |

### session_log (hardening / audit)
MIRRORS the authoritative encrypted + hash-chained JSONL audit trail — **one row
per event** (connect | activity | disconnect) — for SysOp queryability. The
JSONL file is the source of truth; this table is a best-effort mirror.

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| session_id | TEXT | groups a caller's events |
| user_id | FK → user | NULL pre-login |
| username | TEXT | login name if known |
| transport | TEXT | `telnet` \| `ssh` |
| remote_ip | TEXT | |
| event_type | TEXT | `connect` \| `activity` \| `disconnect` |
| action | TEXT | activity name |
| detail | TEXT 🔒 | free-text, sealed at rest |
| minutes | REAL | session duration on disconnect |
| at | TEXT | event timestamp |

## Relationships (at a glance)

```text
user 1───* user_key           (registered SSH keys; soft-revocable)
user 1───* membership
user 1───* message            (author)
user 1───* private_message    (from / to)
user 1───* file_entry         (uploader)
message_area 1───* message
message 1───* message         (parent → replies, threading)
file_area 1───* file_entry
```

---

*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
