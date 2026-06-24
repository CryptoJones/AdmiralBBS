# Changelog

All notable changes to **AdmiralBBS** are documented here.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
versioning: [SemVer](https://semver.org/spec/v2.0.0.html). Per project
convention, **a fix or docs change bumps the PATCH** (`1.0.x`); backward-
compatible features bump the MINOR (`1.x.0`); breaking changes bump the MAJOR.
Each merge to `main` bumps the version (see `version` in
`src/cmd/admiralbbs/main.go`).

## [Unreleased]

_Nothing yet._

## [1.0.2] - 2026-06-24

### Changed
- **Login banner is now generated from the configured BBS name + tagline**
  instead of a hardcoded "AdmiralBBS" logo, so a rebranded BBS shows *its own*
  name on the welcome screen. A SysOp can still supply custom art via `-art`
  (which overrides the generated banner).

### Removed
- The hardcoded "Proudly Made in Nebraska / 🌽" line from the **runtime** welcome
  screen — that's the project's repo signature, not something to stamp on every
  caller's login. It remains in the repo README. (`-art` now defaults to none.)

## [1.0.1] - 2026-06-24

### Added
- **SysOp-customizable branding** — the main menu shows a configurable BBS
  **name** and **tagline** (defaults to AdmiralBBS / its tagline); edit them in
  the control panel (`[X]` → `[S] Branding & MOTD`). Settings live in a new
  `setting` key/value table (migration 010); unset keys fall back to defaults.
- **Message of the Day** — an optional MOTD shown once after login, **before**
  the main menu; the caller must press **SPACE** to continue. A `[O] Message of
  the Day` menu item re-displays it for anyone who blew past it. Edited from the
  same `[S]` SysOp screen.

## [1.0.0] - 2026-06-24

First tagged release — the whole build to date, rolled up. A clean-room,
security-hardened 90s-era ANSI BBS in Go, live on pluto (telnet :1336 apply-only,
SSH :1337 members).

### Core
- Dual transport: Telnet (membership application only) + SSH (members), one
  transport-agnostic session engine; ANSI/CP437 with B&W fallback.
- Encryption in transit (SSH) and **at rest** (XChaCha20-Poly1305; Argon2id key
  from `ADMIRALBBS_KEY`, mlock'd, never on the data volume).
- Tamper-evident audit log (HMAC-SHA256 hash chain) mirrored to `session_log`.
- Pure-Go SQLite (modernc) with WAL; versioned migrations (now at 009).

### Features
- Message boards, private mail, file library (XMODEM up/download, quotas,
  search/sort/filter/delete), all with pagination and per-author edit/delete.
- "New since last visit" board read pointers; who's-online roster.
- Sandboxed door games (subprocess + resident/bridged multiplayer) and the
  SysOp control panel (membership approval + one-time tokens, user/content mgmt,
  audit viewer + chain verify, IP banlist, abuse-report queue).
- Two-factor SSH auth (registered key + password); self-service key & password
  management; user blocking + reporting; rapid-IP-change ("impossible travel")
  flagging.
- **Console Cowboy 2026** — bundled multiplayer cyberpunk MUD door (own engine +
  server; programs/RAM, crews, quests, PvP, multi-stage ICE, leaderboard).

### Auth & sessions (latest)
- **Newest login wins**: a new login displaces an existing session for the same
  handle (instead of being rejected) — intuitive, and self-heals stale sessions.
- **SSH key per tier**: a key may map to at most one SysOp-tier account AND one
  regular account (operator-friendly anti-sockpuppet; relaxed from strict
  one-per-key).
- **Shared SysOp-panel password** (`ADMIRALBBS_SYSOP_PASS`): optional step-up
  secret prompted before the control panel opens.

### Tooling & ops
- `sysopctl` operator tool incl. one-step `bootstrap <handle> <pubkey> [level]`.
- `rekey` key rotation; container image; systemd deployment on pluto.
- `-version` flag.

[Unreleased]: https://github.com/CryptoJones/AdmiralBBS/compare/v1.0.2...HEAD
[1.0.2]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.2
[1.0.1]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.1
[1.0.0]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.0
