# DECISIONS — the house rules

Durable choices future builders must respect. New decisions are appended; old ones are not deleted (they are crossed out and dated if reversed).

## Tech stack

- **Language: Go — because it is memory-safe (eliminates the buffer-overflow class named in the hardening mandate outright), ships a production SSH server in the stdlib-adjacent `golang.org/x/crypto/ssh`, models a multinode BBS naturally as one goroutine per caller, makes door-game sandboxing a matter of `exec.Command` + `SysProcAttr`, and compiles to a single static binary for trivial deployment. (Operator-confirmed 2026-06-23)**
- **Transport: dual listener — Telnet (authentic 90s experience, SyncTERM/NetRunner clients) AND SSH (encrypted, satisfies the hardening mandate, handles terminal resize cleanly). Both feed one shared session/menu engine. (Operator-confirmed 2026-06-23)**
- **Persistence: embedded SQLite (single-file DB, zero-ops, fits "no multi-tenant" scope). Architect default — reversible if scale demands; if reversed, strike this line and record the replacement. (Architect decision 2026-06-23)**
- **Door games run as sandboxed OS subprocesses (separate uid + chroot/jail, no parent-FS access), never in-process — this is the concrete realisation of the sandbox-escape hardening decision below; LBBS uses the same container-isolation approach. (Architect decision 2026-06-23)**
- **Audit logging from day one: every session records remote IP, username (nil pre-auth until Sprint 003), connection time, per-action activity events, and disconnection time. (Operator-directed 2026-06-23)**

## Data layer (decided 2026-06-23)

- **SQLite driver: `modernc.org/sqlite` (pure Go, no cgo) — keeps the single-static-binary promise, cross-compiles trivially, and adds no C boundary (consistent with the memory-safety hardening rationale). (Operator-confirmed)**
- **Journal mode: WAL, plus `busy_timeout=5000`, `foreign_keys=ON`, `synchronous=NORMAL` set as DSN pragmas on every connection — concurrent readers + serialized writer suits the multinode (goroutine-per-caller) model. (Operator-confirmed)**
- **Audit trail is redundant by design: the append-only JSONL file remains the AUTHORITATIVE trail (tamper-evident, survives a DB compromise); a `session_log` table in SQLite MIRRORS it (one row per event) for SysOp queries. JSONL write is the source of truth; the DB mirror is best-effort and never fails a session. (Operator-confirmed)**
- **Data access: hand-written SQL via `database/sql` behind repository interfaces in `src/store/` — no ORM, no query builder. Subsystems depend on repos, never raw SQL. (Architect, operator-confirmed)**
- **Schema migrations: `.sql` files embedded via `go:embed`, applied in order on startup, versioned with `PRAGMA user_version` — no migration framework. (Architect, operator-confirmed)**
- **Password hashing: argon2id (`golang.org/x/crypto/argon2`), stored in the standard encoded string form; never plaintext. (Architect, operator-confirmed)**

## Encryption & transport security (decided 2026-06-23)

- **Encryption in transit: SSH for all member activity; Telnet permitted ONLY for the membership-application screen, then the applicant is told to reconnect via SSH. No password or secret ever crosses a plaintext channel — applicants set their password on first SSH login. (Operator-confirmed)**
- **Encryption at rest, two layers: (1) app-level AEAD — XChaCha20-Poly1305 (`golang.org/x/crypto/chacha20poly1305`) seals sensitive payloads (message/mail bodies, file-library blobs, PII, audit content); (2) the whole data directory runs on an encrypted volume. (Operator-confirmed)**
- **Master key: derived (Argon2id) from a startup secret `ADMIRALBBS_KEY` supplied via env var / Docker secret, never written to the data volume, never entered in chat (per credential-handling rule). Held only in memory, `mlock`'d, zeroed on exit. A non-secret KDF salt persists in the data dir. (Operator-confirmed)**
- **The daemon refuses to start without `ADMIRALBBS_KEY` — encryption is mandatory, not optional. (Architect)**

## Security hardening — foundational set (decided 2026-06-23, from the 100k-ft review; see planning/RISKS.md)

- **SEC-1 Door isolation: doors run as sandboxed subprocesses with a SCRUBBED environment (master key + all secrets stripped before exec), separate uid + jail, dropped capabilities, rlimits, wall-clock timeout, no inherited fds, no network. (Decision; realised S007)**
- **SEC-2 Membership claim: the "set password on first SSH login" step requires a one-time approval token delivered out-of-band (or an SSH public key captured at application time) to close the account-takeover window. (Decision; realised S003)**
- **SEC-3 DoS limits: max concurrent sessions, per-IP connection throttle, SSH handshake timeout, idle timeout, and slow-loris read deadlines, enforced in the transport/session layer. (Decision; foundation)**
- **SEC-4 Auth defence: rate-limit + exponential backoff/lockout on BBS login, a generic "login failed" message, and constant-time user lookup to prevent enumeration. (Decision; realised S003)**
- **SEC-5 Output sanitisation: user-generated content (message bodies, handles) is escape-sanitised on OUTPUT before display to other callers, not just on input. (Decision; foundation + S004)**
- **SEC-6 Audit integrity: the audit trail is confidentiality + integrity — each sealed line carries an HMAC hash-chain committing to the previous line, so any edit/truncation is detectable. (Decision; foundation)**
- **SEC-13 Telnet pubkey-paste MITM: applicants paste their SSH public key(s) over plaintext Telnet (public keys are not secret). Integrity risk (a MITM could swap the key) is mitigated by the SysOp confirming the key fingerprint out-of-band before approval. (Decision; S003)**

## Authentication & membership (decided 2026-06-23)

- **Two-factor SSH for members: after onboarding, every connection requires BOTH a registered SSH public key (offered key must match one of the user's ACTIVE keys) AND the BBS password (argon2id), over the encrypted channel. Telnet is apply-only and never authenticates a member. (Operator-confirmed; enforced S003)**
- **Multiple keys per user: users register one or more SSH public keys at application and can add / revoke keys later (revocation is soft — the record is kept, `revoked_at` set). Stored in the `user_key` table. Self-service management in S003; SysOp oversight in S008. (Operator-confirmed)**
- **One-time approval token delivery: delivered OUT-OF-BAND by the SysOp; NO Signal/Telegram/email integration is baked into the BBS (that would add network egress + a third-party dependency + a secret, against the hardened posture). The application collects an optional contact field (and, recommended for this key-managing audience, an optional PGP public key / fingerprint). The SysOp Control Panel shows the token + contact + SSH key fingerprint; the SysOp relays it through whatever channel they already trust — which also confirms the key fingerprint (SEC-13). Token is single-use, time-limited, stored hashed (never plaintext). Recommended out-of-band channel: PGP-encrypted email (end-to-end, uses existing mail infra, no new SaaS). If delivery is ever automated, do it as an optional pluggable notifier and automate PGP-email (`gpg` + SMTP) — NOT a chat bot. (Operator-confirmed 2026-06-23)**

- **Daily time budget: default 60 minutes/day per member, configurable per-user (`user.daily_minutes`) and via a server default flag; SysOps (access level ≥ 100) are unlimited. Architect default (classic BBS time limit) — reversible; CryptoJones can set another number. (Architect 2026-06-23)**

## Deployment (decided 2026-06-23)

- **Runs natively OR in a container. A multi-stage Dockerfile builds the static binary and ships it on a minimal base; `docker-compose.yml` wires ports (telnet 2323, ssh 2222) and a volume for the SQLite DB / audit log / SSH host key so state persists across container restarts. The native path (`go build` + run) stays first-class. (Operator-directed 2026-06-23)**

## Decisions captured during Sprint 001 discovery

- **The BBS must have ansi graphics when availble to the user, but must support older terminal types in black and white (2026-06-23)**
  - _Realisation:_ follow the ANSI-BBS spec — assume 80 columns + CP437, detect terminal capability on connect, and silently ignore unsupported escape sequences (graceful degrade to plain text). Never assume row count.
- **The bbs must be security hardened against buffer overflows, packet ejection and sandbox escape tactics. (2026-06-23)**
  - _Realisation:_ memory-safe language (Go) kills buffer overflows; all caller input is length-bounded and validated before parse ("packet ejection"/injection); door games run sandboxed (see above); the daemon runs as a non-root user.
- **The BBS must have a user message board, a file library, door games, and private messaging features. (2026-06-23)**

## Explicitly out of scope

- multi-tenant
- web version

## How to add a decision

When something gets decided in conversation, append it to the list above in the same format. **Always include the date** — `socrates timeline` reads the trailing `(YYYY-MM-DD)` to surface decisions chronologically:

```
- **<choice> — because <reason> (YYYY-MM-DD)**
```

If the decision is reversed, do not delete the line. Strike it through with `~~...~~` and add the new decision below with the date.
