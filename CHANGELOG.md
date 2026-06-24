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

## [2.0.0] - 2026-06-24

**The bundled door game is gone — extracted to its own repo.** Chrome Circuit
Cowboys now ships and versions independently at
[CryptoJones/ChromeCircuitCowboys](https://github.com/CryptoJones/ChromeCircuitCowboys)
(first standalone release v1.0.0). AdmiralBBS is now a pure BBS: it ships no
game code, only the generic resident-door framework any door can plug into.

### Removed (breaking)
- All bundled game code (`src/game/cowboy`, `src/cmd/cowboy`) and its tests. The
  door is no longer built, shipped, or versioned with the BBS.
- The hardcoded `-cowboy <addr>` flag. **Migration:** register the door with the
  new generic `-door` flag (below) — e.g.
  `-door "Chrome Circuit Cowboys|tcp|127.0.0.1:4000|0"`.

### Added
- **Generic resident-door registration.** Repeatable `-door
  "name|network|address|minlevel"` flag registers any door speaking the
  resident-door bridge protocol — nothing game-specific is compiled in.
- **Forge-agnostic update check.** `-update-url` (or `$ADMIRALBBS_UPDATE_URL`)
  points at a forge `releases/latest` JSON endpoint (GitHub/Codeberg/Forgejo all
  share the `{"tag_name":"vX.Y.Z"}` shape); on startup the BBS logs a notice if a
  newer release exists. No forge is hardcoded — empty URL = no check.
- **`scripts/install-door.sh`** — fetches, builds, and installs a resident door
  from a configurable repo (defaults to Chrome Circuit Cowboys; override
  `DOOR_REPO`/`DOOR_REF`/etc. for Codeberg or any other forge).
- **`-version`** flag prints the version and exits.

## [1.6.0] - 2026-06-24

### Changed
- **The door is now a fully generic cyberpunk RPG — rebranded "Chrome Circuit
  Cowboys" (C³) and scrubbed of all franchise-specific IP.** No term is tied to
  any one property:
  - **Title:** Console Cowboy 2026 → **Chrome Circuit Cowboys** (banner, help,
    leaderboard, door registration, server log). Internal package/binary/service
    names kept as `cowboy` (not player-facing).
  - **Classes:** Netrunner/Solo/Fixer/Techie → **Hacker/Enforcer/Operator/Mechanic**
    (same stat profiles).
  - **Currency:** "eddies" → **"scrip"** (display; the €$ symbol stays).
  - **Re-sleeve / cortical stack / sleeve** (Altered Carbon) → **re-clone / neural
    backup / body**; "Re-Sleeve Bay" → **"Re-Clone Bay"**.
  - **NCPD** → a generic **City Security drone**; **Arasaki Plaza** → **Corporate
    Plaza**; **The Sprawl** → **The Strip**; **ripperdoc** → **Emergency Medic
    (EM)** (re-installs cyberware at the Night Market); plus assorted flavor.

### Added
- **RP emotes.** `emote <action>`, `me <action>`, and `:<action>` broadcast a
  third-person action to the room (e.g. `me lights a cig` → "Wintermute lights a cig").

## [1.5.0] - 2026-06-24

### Changed
- **Console Cowboy: PvP is now live EVERYWHERE except the safe zone outside the
  clone pods.** Previously PvP was Net-only. Now you can `attack <runner>` in any
  room (meatspace duels use your melee swing; Net duels still breach + spend RAM).
  - **No-violence zone:** Neon Alley (the street the clone pods open onto) and the
    private Re-Sleeve Bay are safe. **Draw on another runner there and an NCPD
    security drone flatlines _you_** on the spot (you drop your sleeve + pay the
    clone fee; the target is untouched).
  - A duel ends if either runner leaves to a safe zone.

## [1.4.0] - 2026-06-24

### Added
- **Console Cowboy: corpse loot + ripperdoc cyberware re-install (Altered-Carbon
  death, part 2).** When you flatline, your **old sleeve drops as a lootable
  corpse** right where you fell, holding **all your inventory items plus your
  cyberware** (weapon + cyberdeck) — which is **stripped from the fresh clone**.
  - `loot` strips every flatlined sleeve in the room into your pack. **Open
    recovery:** anyone can loot any sleeve (recover for a crewmate — or swipe it).
    The corpse persists until looted.
  - Recovered **cyberware must be re-installed at a ripperdoc** (`install <cyber>`,
    at the **Night Market**) to work again; consumables are usable immediately.
  - `give <item> <runner>` hands recovered gear back to a crewmate in the room.
  - Corpses are in-memory world state (not persisted across a server restart).

## [1.3.0] - 2026-06-24

### Changed
- **Console Cowboy: re-sleeve death model + spawn-safe clone bay (Altered-Carbon
  style).** Foundation pass:
  - New/respawning runners now wake in a **private, isolated Re-Sleeve Bay** (the
    clone clinic), not a shared street room — so a respawn can't be spawn-camped.
    Step `out` into the street; `home`/`in` returns you (from the street only —
    not a teleport, and combat-blocked).
  - Death is now a **re-sleeve**: your stack restores into a **fresh, full-HP
    clone**. **No XP/skill loss** (the stack is backed up). The only cost is a
    **10% clone-body fee** of your credits — credits are never otherwise reduced.
  - **Leadership passes on death:** if a crew leader flatlines, the crew passes to
    the longest-tenured surviving member (a dead runner doesn't keep leading).
  - _Next (v1.4.0): your old sleeve drops as a lootable corpse holding your gear +
    cyberware; cyberware must be re-installed at a ripperdoc._

## [1.2.0] - 2026-06-24

### Changed
- **Console Cowboy: crews are now consent-based with a leader.** Previously
  `group <runner>` *conscripted* the target into your crew with no say — anyone
  could force-group anyone. Now:
  - `invite <runner>` (or `group <runner>`) sends an **invite**; the target must
    `accept` (or `decline`). No one is added without consent.
  - Only the **crew leader** can invite. A solo runner forming a crew becomes its
    leader; leadership passes to the next member if the leader leaves.
  - `group`/`crew` shows the crew (leader marked); pending invites expire when
    either party jacks out.
  Found in live co-op play.

## [1.1.2] - 2026-06-24

### Fixed
- **Console Cowboy: welcome-banner box border is now aligned.** The box rows were
  hand-spaced to mismatched widths (title 2 cols short, tagline 2 cols long — the
  em-dash `—` made it worse), so the right `║` zig-zagged and didn't meet the
  `╗`/`╝`. The banner is now sized to its widest line and each row padded by rune
  count, so the borders line up on any terminal. Found in live co-op play.

## [1.1.1] - 2026-06-24

### Fixed
- **Console Cowboy: `use` no longer wastes a consumable at full HP/RAM.** Using a
  stimpak at full HP (or a ram-chip at full RAM) consumed the item for zero
  benefit; it's now refused with "already full — save the …" and the item is
  kept. Found in local playtesting.
- **Console Cowboy: `use` with no argument** said "You don't have a ." — it now
  prompts "Use what? (see INVENTORY)".

## [1.1.0] - 2026-06-24

### Added
- **Paged audit log viewer.** The SysOp audit log was a single fixed screen of
  the 20 newest events with no way to look further back. It is now an interactive
  pager: `[N]ext` page, `[P]rev`, `[F]` jump +10 pages, `[R]` jump −10 pages
  (both clamped to the available range), and `[Q]` to exit. The header shows the
  current page, total pages, and the event range. The JSONL chain verification
  and rapid-IP-change summaries are computed once on entry (not re-run per page),
  so navigation stays fast. New `store.SessionLog.Page(limit, offset)` backs it.

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

[Unreleased]: https://github.com/CryptoJones/AdmiralBBS/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v2.0.0
[1.6.0]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.6.0
[1.5.0]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.5.0
[1.4.0]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.4.0
[1.3.0]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.3.0
[1.2.0]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.2.0
[1.1.2]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.1.2
[1.1.1]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.1.1
[1.1.0]: https://github.com/CryptoJones/AdmiralBBS/releases/tag/v1.1.0
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
