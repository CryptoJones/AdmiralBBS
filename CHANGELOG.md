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

## [1.0.9] - 2026-06-24

### Fixed
- **Console Cowboy: `inventory` now shows your credits.** Eddies (€$) only
  appeared on the `score` sheet; they now head your inventory listing too, where
  players expect to see their money.

## [1.0.8] - 2026-06-24

### Fixed
- **Console Cowboy: combat output no longer eats your typing.** Async output
  (combat/chat/room events) interleaved with the caller's in-progress input,
  garbling commands like `flee`. The server now uses a managed prompt: each async
  line wipes the current input row, prints, then redraws the prompt with the
  caller's partial input intact (engine routes the status prompt through a
  dedicated `prompter` sink). A full-screen scroll-region layout is the next step.

## [1.0.7] - 2026-06-24

### Added
- **Console Cowboy autosave + save-on-shutdown.** The world goroutine now
  autosaves every connected player every 30s, and a SIGINT/SIGTERM handler
  flushes all players before the process exits (`World.SaveAll`). Previously
  characters persisted only on a clean disconnect, so a server restart/crash
  lost progress since login. After this lands, cowboy restarts are non-destructive.

## [1.0.6] - 2026-06-24

### Fixed
- **Branding/MOTD edits take effect immediately, without a relog.** The main
  menu was built once at login, snapshotting the BBS name/tagline/banner and the
  MOTD item; it now carries a `Refresh` hook that re-reads settings on every
  render, so a SysOp's marketing changes show on the next screen draw.

## [1.0.5] - 2026-06-24

### Fixed
- **Console Cowboy: the multi-stage Gauntlet ICE respawned into the void.** After
  it was beaten, the mob template stayed mutated to its final form (which has no
  home room), so it respawned as the lethal lock into room "" and never returned
  to the Sentinel Lattice. Mobs now remember their origin template and reset to
  it on respawn. (Regression test beats the gauntlet and asserts the first form
  reappears in the Lattice.)

## [1.0.4] - 2026-06-24

### Fixed
- **Console Cowboy door: spurious extra prompt / input stall.** The terminal
  line reader peeked for a CRLF partner with a *blocking* read, so an
  interactive lone CR/LF (one keystroke at a time) could strand a half-pair that
  the next read returned as a blank line — printing a second prompt — or block
  waiting for a partner byte that never arrived. It now only consumes a paired
  terminator when one is already buffered. (Batch tests never hit this because
  they send all bytes at once; added an `io.Pipe` regression test that does.)

## [1.0.3] - 2026-06-24

### Fixed
- Two more hardcoded "AdmiralBBS" strings now use the **configured BBS name**:
  the logoff message ("Thanks for calling <name>. NO CARRIER") and the Telnet
  membership-application screen title. Branding is now consistent everywhere a
  caller sees it. (`RunApply` now takes the store so it can read settings.)

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

[Unreleased]: https://github.com/CryptoJones/AdmiralBBS/compare/v1.0.9...HEAD
[1.0.9]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.9
[1.0.8]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.8
[1.0.7]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.7
[1.0.6]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.6
[1.0.5]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.5
[1.0.4]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.4
[1.0.3]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.3
[1.0.2]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.2
[1.0.1]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.1
[1.0.0]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.0.0
