# STATE — current moment

_Last updated: 2026-06-23_

## Active sprint

**Autonomous loop: "implement all the bbs features"** (started 2026-06-23).
Landed on `main`: S002 spine, encrypted data layer, S003 2FA auth, S004
message boards, S005 private mail, **S006 file library** (areas, upload/download,
blobs sealed at rest + id-based paths (SEC-7), size cap, access gating). Next:
S007 door games → S008 SysOp control panel → S009 hardening/deploy.

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
