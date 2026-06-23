# SprintPlanning — AdmiralBBS implementation roadmap

> **Sprint 001 acceptance deliverable.** A plan to implement *each part* of the
> BBS. Stack and transport are locked (see `planning/DECISIONS.md`): **Go**,
> dual **Telnet + SSH** transport, embedded **SQLite**, sandboxed-subprocess
> door games. Each sprint below gets its own `planning/sprints/NNN-name/`
> folder when it becomes active. Sprint 002 is fully specced now.

## Sequencing principle

Build the **spine first** (transport → session → terminal → menu), because
every other feature is written against the `Session` and menu engine. Then add
subsystems in order of dependency, hardening continuously rather than as a
bolt-on at the end.

## The parts, and the sprint that delivers each

| # | Sprint | Delivers | Depends on |
|---|---|---|---|
| 002 | **Core session engine** | Telnet + SSH listeners, `Session`, hardened input reader, terminal detection (ANSI/BW, CP437, dims), ANSI/B&W screen writer, menu engine, `.ans` loader. A caller can connect on both transports and navigate a static menu. | — |
| 003 | **Users & membership** | Registration, password hashing, login, access levels, **manual membership approval** workflow, **daily time budget** enforcement, SQLite store + repos. | 002 |
| 004 | **Message boards** | Message areas, post/read/reply, threading, per-area access gating. | 002, 003 |
| 005 | **Private messaging** | User-to-user mail: compose, inbox, read/unread, reply. | 003 |
| 006 | **File library** | File areas, listings, descriptions, download (Zmodem or HTTP-side-channel), upload, access gating. | 003 |
| 007 | **Door games** | Sandboxed-subprocess launcher, dropfile generation (`door32.sys`/`DOOR.SYS`), session I/O piping, door registry. | 002, 003 |
| 008 | **SysOp Control Panel** | SSH-only, access-level ≥80 admin console: membership queue, user management, area/door management, audit-log viewer + chain verification, live session/system status, encryption-key status. | all |
| 009 | **Hardening pass & deploy** | Fuzz/negative-test sweep against the full hardening matrix, container hardening (read-only FS, seccomp, dropped caps), `govulncheck` in CI, key-rotation runbook. | all |

Each sprint's exit gate is in `docs/VALIDATION.md`: real SyncTERM/NetRunner
confirmation for interactive features, and a passing negative test for any
attack surface introduced.

## Security hardening map (see planning/RISKS.md, DECISIONS.md)

The 100k-ft review's findings are owned by sprints, not deferred to a vague
"hardening later":

| Item | Concern | Lands in |
|---|---|---|
| Encryption at rest / in transit | two-layer AEAD + encrypted volume; SSH-only members, Telnet=apply | **foundation** (data layer + transport), this branch |
| SEC-3 | DoS limits (session caps, per-IP throttle, timeouts) | **foundation** (transport/session) |
| SEC-5 | output escape-sanitisation of stored content | **foundation** (screen) + S004 |
| SEC-6 | audit confidentiality + HMAC hash-chain integrity | **foundation** (audit) |
| SEC-1 | door subprocess isolation + scrubbed env | S007 |
| SEC-2 | one-time approval token (no takeover window) | S003 |
| SEC-4 | login backoff/lockout, generic errors, constant-time lookup | S003 |
| SEC-7 | file-library path traversal, quotas, zip-bombs | S006 |
| SEC-8 | server-side authz on every action; safe SysOp bootstrap | S003+ |
| SEC-9 | minimise PII collected over Telnet | S003 |
| SEC-10/11/12 | container hardening, govulncheck/CI, key strength + rotation | deploy / CI |

## Per-part build notes

### 002 — Core session engine (the spine)
- `src/transport/`: `net_telnet` (IAC negotiation, NAWS) and `net_ssh`
  (`x/crypto/ssh`, host key, pty-req/window-change). Both yield a common
  `Conn`.
- `src/session/`: bounded/validated input reader (the hardening boundary),
  terminal capability struct, time-budget hook (enforced in 003).
- `src/screen/` + `src/menu/`: capability-aware ANSI writer that degrades to
  B&W on one code path; CP437 `.ans` loader; data-driven menus + dispatcher.
- **Done when:** connect via SyncTERM over telnet *and* ssh, see an ANSI
  welcome screen, navigate a static menu; same screen is readable on a plain
  B&W terminal; fuzzing the reader never crashes the process.

### 003 — Users & membership
- `src/store/` SQLite + repositories per `docs/DATA_MODEL.md`.
- Argon2/bcrypt hashing; never store plaintext.
- Pending users held at a holding screen; SysOp approve/deny.
- Daily-minutes budget decremented from `session_log`; kick on exhaustion.
- **Open questions to resolve before this sprint:** default `daily_minutes`;
  exact approval workflow (auto-email the SysOp? in-BBS queue?). See
  `planning/QUESTIONS.md`.

### 004 — Message boards
- Areas with `min_access_level`; threaded messages via `parent_id`.
- Read/post/reply screens through the menu engine.

### 005 — Private messaging
- Inbox with unread counts; compose/reply; read receipts (`read_at`).

### 006 — File library
- Areas + entries; on-disk file paths kept outside the DB.
- Transfer mechanism is a sub-decision for that sprint (classic Zmodem vs. a
  simpler modern path) — record it in `DECISIONS.md` when chosen.

### 007 — Door games
- Launcher spawns the door as a **separate uid inside a chroot/jail**, pipes
  session I/O, writes the dropfile the door expects. This *is* the
  sandbox-escape mitigation — validate with a door that tries to escape.

### 008 — SysOp Control Panel
The operator's command center — an in-BBS admin console reached from the main
menu, **SSH-only and gated to access-level ≥80** (SysOp/Co-SysOp, per
`docs/PERMISSIONS.md`). Server-side authz on every action (SEC-8), never just a
hidden menu. Panels:

- **Membership queue** — list pending applications (with the applicant's note),
  approve/deny, and on approval **issue the one-time token** the applicant uses
  to set their password on first SSH login (SEC-2). Deny with a reason.
- **User management** — search/list users; view profile (decrypts PII on the
  fly); set access level; suspend/reinstate; set per-user daily-minutes; clear a
  password to force re-onboarding. Never displays password hashes.
- **Content management** — create/edit/delete message areas and file areas;
  register/remove door games and their sandbox config (SEC-1).
- **Audit log viewer** — query the `session_log` mirror (by user, IP, session,
  time window) and **verify the hash-chain** of the authoritative JSONL trail
  (`audit.ReadAll`/chain check), surfacing any tampering (SEC-6).
- **Live status** — active sessions/nodes, per-IP connection counts, DoS-limit
  headroom (SEC-3), and read-only encryption status (key loaded, salt present,
  "no plaintext at rest" confirmation).
- **Broadcast** — optional message-of-the-day / node broadcast.

Build notes: reuse the menu engine; all panels are repos + actions over the
existing `store`. The audit viewer is the first consumer of `audit.ReadAll`.
This sprint depends on every subsystem existing, hence near the end.

### 009 — Hardening pass & deploy
- Full fuzz/negative-test sweep against the `docs/VALIDATION.md` hardening
  matrix (incl. the telnet/SSH protocol parsers, not just the input sanitiser).
- Container hardening: read-only root FS, `no-new-privileges`, dropped caps,
  seccomp profile, Docker secrets for the key (SEC-10).
- `govulncheck` in CI; dependency pinning; image scanning (SEC-11).
- Key-strength guidance + a documented re-encrypt/rotation runbook (SEC-12).
- Confirm non-root deployment end to end.

---

*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
