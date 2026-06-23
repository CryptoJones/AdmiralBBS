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
| 008 | **SysOp tools & hardening pass** | Admin menus (approve members, manage areas/doors, read audit log), fuzz/negative-test sweep against the full hardening matrix, non-root deployment hardening. | all |

Each sprint's exit gate is in `docs/VALIDATION.md`: real SyncTERM/NetRunner
confirmation for interactive features, and a passing negative test for any
attack surface introduced.

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

### 008 — SysOp tools & hardening pass
- Admin surface for everything in `docs/PERMISSIONS.md` level 80–100.
- Full fuzz/negative-test sweep against the `docs/VALIDATION.md` hardening
  matrix; confirm non-root deployment.

---

*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
