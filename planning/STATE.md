# STATE — current moment

_Last updated: 2026-06-24_

> **What's left lives in [`planning/BACKLOG.md`](BACKLOG.md)** — now FULLY
> CLEARED (2026-06-24). All P1/P2/P3 items shipped across S014–S021.

**S014–S021 (backlog clear-out, 2026-06-24)** — worked the whole BACKLOG top to
bottom, each item tested and landed on `main` with a verified-green CI run:
- S014 SSH-key fingerprint uniqueness (one account per key)
- S015 SysOp IP banlist + transport-layer enforcement (drops at accept)
- S016 user-to-user moderation (block/ignore + report-to-SysOp)
- S017 pagination (mail / boards / files)
- S018 message + mail edit/delete
- S019 "new since last visit" read pointers
- S020 who's-online (live roster)
- S021 rapid-IP-change ("impossible travel") flagging (visibility only)
Schema is at migration 008. CI green @ `5f4bad4`.

**S022 (Console Cowboy 2026 — multiplayer cyberpunk MUD, 2026-06-24)** — a new
resident door game. Standalone engine (`src/game/cowboy`) + TCP server
(`src/cmd/cowboy`); BBS bridges callers in via `-cowboy <addr>`. MajorMUD-style:
shared world, rooms (Night City + the Net), to-hit-vs-AC combat on a tick,
XP/level (cap 50)/€$ economy, shops, character creation (4 classes + skill
points), and fixer bounty quests. Netrun twist: in the Net, ATTACK breaches ICE
with Intelligence. Mechanics grounded in research (MajorMUD/LORD + CP2020/GURPS).
Load-tested at 50 concurrent unique players in one world. See
[`docs/CONSOLE_COWBOY.md`](CONSOLE_COWBOY.md).

**S023 (Console Cowboy — netrun depth, 2026-06-24)** — three follow-ons:
RAM/cyberdeck resource for breaching (breaches spend RAM; sputter when empty;
ram-chips refill, cyberdeck raises the cap; shown in the NET prompt); a deeper
Net (Grid Node → Sentinel Lattice → Black ICE Fortress) with a **multi-stage
Gauntlet ICE** that morphs white shell → black core → lethal lock; and **PvP
netrunning** — duel other runners in the deep Net, loser flatlines and winner
siphons eddies. All persisted and tested.

**S013 (file-library hardening + CI green)** — landed on `main @ e9d0dfd`.
Atomic per-user upload quota (closes the TOCTOU race), duplicate-filename
rejection, file-area search/sort/filter/delete, XMODEM upload byte-cap, and
self-service password change. **CI is GREEN for the first time** (build / vet /
test / govulncheck on the Linux runner) — it had been red since S009 over a
non-portable path-traversal test assertion, now fixed. Remaining work is
catalogued in BACKLOG.md.

## Active sprint

**COMPLETE (2026-06-23).** All sprints landed on `main`: S002 spine, encrypted
data layer, S003 2FA auth, S004 boards, S005 mail, S006 files, S007 sandboxed
doors, S008 SysOp control panel, S009 hardening/deploy (CI + govulncheck clean +
hardened container + ops/key-rotation runbook). All four founding features, the
operator console, encryption in transit + at rest, and the SEC-1…13 register are
realised. S010 (feature polish): per-user upload quotas, key rotation (cmd/rekey), door
uid/namespace isolation (Linux), XMODEM transfer.

**S011 (verification & fixes)** — a re-audit caught that "complete" was
overclaimed: fixed a CRITICAL door-exit hang (session froze when a door exited),
added the missing SysOp bootstrap (`cmd/sysopctl` — the panel was unreachable),
added graceful shutdown, and added the integration coverage that was missing
(`TestFullMemberJourney` driving the real menus end-to-end + door-through-launcher
e2e + telnet IAC fuzz). Lesson: ship an end-to-end test of the real journey, not
just unit tests, before calling anything done.

**S012 (multi-user correctness & door models)** — more gaps CJ surfaced: per-user
session cap (session.Presence — a user could multiply their daily budget by logging
in 50× at once); node pool (unique node per session); all three door models now
(single-player, turn-based file-shared via $DOORSHARE, and RESIDENT real-time
multiplayer MajorMUD-style via doors.Bridge + a 'resident' door kind, migration
003); persistent door working dirs; and message-board search / sort-by-date /
filter-by-user (search decrypts since content is sealed). Remaining (flagged):
pagination, delete/edit, unread pointers, who's-online. Original history below.

## Autonomous loop history

**"implement all the bbs features"** (started 2026-06-23).
Landed on `main`: S002 spine, encrypted data layer, S003 2FA auth, S004
message boards, S005 private mail, S006 file library, **S007 door games**
(sandboxed-subprocess launcher: scrubbed env so the master key never leaks
into a door, jail dir, CPU rlimit, wall-clock timeout + process-group kill,
door32.sys dropfile, bundled demo door — SEC-1), **S008 SysOp control panel**
(membership approval + token issuance, user management, content management,
audit viewer + chain verification; access ≥80, server-side gated). All four
founding features + the operator console are done. Last: S009 hardening/deploy
(container hardening, govulncheck/CI, key-rotation runbook).

## (earlier) Active sprint

**003 — Users & Membership (2FA auth)** — landed on `main`.

Data layer + encryption landed on `main` (merge `55ef243`). On the S003 branch:
one-time approval tokens (hashed, single-use, 72h TTL), `Users.Approve`, SSH
two-factor enforcement (PublicKeyCallback gates the registered key; password is
the 2nd factor), first-login onboarding (token → set password), login backoff +
generic errors (SEC-4), daily time-budget enforcement, and self-service SSH-key
management. `go build/vet/test` green; SSH 2FA smoke-tested (reject-without-key;
onboard-with-key). Remaining S003: message-board-independent polish; full SysOp
approval UI is S008. Earlier sprint context below.

## (prior active sprint)

**002 — Core Session Engine**

## Status

**Branch `feat/data-layer`** (not yet committed/pushed) adds, on top of the
Sprint 002 spine: the encrypted data layer (modernc SQLite + WAL, migrations,
repos for users/keys/memberships, argon2id), the `crypto.Vault` (Argon2id key
from `ADMIRALBBS_KEY`, XChaCha20-Poly1305 at rest, mlock'd), dual audit
(encrypted + HMAC hash-chained JSONL, mirrored to `session_log`), foundational
hardening (DoS limits, idle timeout, output sanitisation), Telnet=apply-only
with multi-SSH-key collection, two-factor-SSH data shape, and containerisation
(Dockerfile/compose). `go build/vet/test` green; daemon refuses without the key;
telnet apply + ssh paths smoke-tested; sensitive fields verified ciphertext at
rest. Full 2FA enforcement + key-management UI + membership approval land in
Sprint 003 / the SysOp Control Panel (S008).

---
_Earlier:_ Sprint 001 complete. **Sprint 002 (Core Session Engine) implemented and
validated** (2026-06-23): Go module, Telnet (`:2323`) + SSH (`:2222`)
listeners feeding one transport-agnostic `Session`, hardened input sanitiser,
terminal detection (ANSI/B&W, CP437), capability-aware screen writer, data-
driven menu engine, and the operator-requested **audit trail** (IP, username,
connect time, activities, disconnect time + duration). `go build`/`go test`
green; sanitiser fuzzed 1M+ execs with no crash; both transports smoke-tested
on the wire (telnet ANSI render + SSH with username capture).

## Next action

Operator review of the Sprint 002 spine. Then resolve the two open questions
that gate Sprint 003 (default daily-minutes budget; membership-approval
workflow) and begin Sprint 003 (Users & membership + SQLite store).

## Blockers

- **Sprint 003 is gated** on two open questions: default daily time budget and
  the membership-approval workflow (see `planning/QUESTIONS.md`).

## Recently completed

- Project scaffolded via 120xSocrates (2026-06-23).
- Initial planning interview captured into this folder.
- **Sprint 001 closed (2026-06-23):** stack/transport decided with operator;
  `docs/ARCHITECTURE.md`, `docs/DATA_MODEL.md`, `docs/VALIDATION.md`,
  `docs/PERMISSIONS.md` populated; `planning/SprintPlanning.md` roadmap written;
  Sprint 002 folder created.
