# Data Model — AdmiralBBS

> Embedded SQLite. Entities below are the first-draft schema; the Builder
> refines column types per sprint. Repository interfaces in `src/store/` wrap
> these so subsystems never write raw SQL.

## Entities

### user
The caller account. Membership status gates access (see `membership`).

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| handle | TEXT UNIQUE | the BBS alias |
| password_hash | TEXT | bcrypt/argon2 — **never** plaintext |
| real_name | TEXT | optional |
| email | TEXT | optional |
| access_level | INTEGER | SysOp / Co-SysOp / Member / Guest (see `docs/PERMISSIONS.md`) |
| status | TEXT | `pending` \| `approved` \| `denied` \| `suspended` |
| daily_minutes | INTEGER | per-day time budget (open question — default TBD) |
| created_at | TIMESTAMP | |
| last_login_at | TIMESTAMP | |

### membership
The manual-approval workflow for new applicants (open question in QUESTIONS.md).

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| user_id | FK → user | |
| applied_at | TIMESTAMP | |
| reviewed_by | FK → user | the SysOp who acted |
| reviewed_at | TIMESTAMP | |
| decision | TEXT | `pending` \| `approved` \| `denied` |
| note | TEXT | SysOp reason |

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
Append-only record of connections — supports the audit posture.

| Field | Type | Notes |
|---|---|---|
| id | INTEGER PK | |
| user_id | FK → user | NULL for failed pre-auth |
| transport | TEXT | `telnet` \| `ssh` |
| remote_addr | TEXT | |
| connected_at | TIMESTAMP | |
| disconnected_at | TIMESTAMP | |
| minutes_used | INTEGER | feeds the daily budget |

## Relationships (at a glance)

```text
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
