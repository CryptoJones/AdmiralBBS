# DECISIONS — the house rules

Durable choices future builders must respect. New decisions are appended; old ones are not deleted (they are crossed out and dated if reversed).

## Tech stack

- **Language: Go — because it is memory-safe (eliminates the buffer-overflow class named in the hardening mandate outright), ships a production SSH server in the stdlib-adjacent `golang.org/x/crypto/ssh`, models a multinode BBS naturally as one goroutine per caller, makes door-game sandboxing a matter of `exec.Command` + `SysProcAttr`, and compiles to a single static binary for trivial deployment. (Operator-confirmed 2026-06-23)**
- **Transport: dual listener — Telnet (authentic 90s experience, SyncTERM/NetRunner clients) AND SSH (encrypted, satisfies the hardening mandate, handles terminal resize cleanly). Both feed one shared session/menu engine. (Operator-confirmed 2026-06-23)**
- **Persistence: embedded SQLite (single-file DB, zero-ops, fits "no multi-tenant" scope). Architect default — reversible if scale demands; if reversed, strike this line and record the replacement. (Architect decision 2026-06-23)**
- **Door games run as sandboxed OS subprocesses (separate uid + chroot/jail, no parent-FS access), never in-process — this is the concrete realisation of the sandbox-escape hardening decision below; LBBS uses the same container-isolation approach. (Architect decision 2026-06-23)**
- **Audit logging from day one: every session records remote IP, username (nil pre-auth until Sprint 003), connection time, per-action activity events, and disconnection time. In Sprint 002 (no DB yet) these append to a structured JSONL audit file + stdout; in Sprint 003 they migrate to the `session_log` / activity tables in SQLite. (Operator-directed 2026-06-23)**

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
