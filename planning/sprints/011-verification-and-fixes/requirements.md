# Sprint 011 — Verification & Fixes | requirements & acceptance

## Why
A 100,000-ft re-audit (prompted by CryptoJones) found that "COMPLETE / all green"
was overclaimed: the admin panel was unreachable, a door froze the session on
exit, and the interactive menus had no integration coverage. This sprint fixes
those and adds the missing end-to-end verification.

## Fixed
- **CRITICAL — door froze the session on exit.** With the session as `cmd.Stdin`,
  `cmd.Wait` blocked forever on the stdin-copy goroutine after the door process
  exited (`WaitDelay` didn't help). Caught by a new end-to-end launcher test
  (door never returned). Fix: give the child stdin via an `*os.File` pipe and
  forward input ourselves; `Launch` now returns immediately on door exit.
- **CRITICAL — no SysOp bootstrap.** Access level was only set inside the
  ≥80-gated panel, so no one could ever become SysOp on a fresh install. Added
  `cmd/sysopctl` (`list` / `approve <handle> [level]` (+token) / `promote`).
  Documented in `docs/OPERATIONS.md`.
- **No graceful shutdown.** `main` now flushes audit + closes DB/vault on
  SIGINT/SIGTERM (container stop).

## Added (the missing verification)
- **`TestFullMemberJourney`** — drives the real menus end to end: telnet apply →
  approve + token → SSH onboarding → board post → mail send → file upload, with
  store side-effects asserted. (The feature menus had zero interaction coverage.)
- **`TestDemoDoorPlaysThroughLauncher`** — the bundled door actually plays through
  the sandbox launcher and Launch returns cleanly.
- **Telnet IAC parser fuzz** (`FuzzTelnetFeed`) — a claimed-but-never-done item;
  2.3M execs, no crash.

## Honest status correction
"Login lockout" is in-session backoff only (no persistent cross-reconnect
lockout) — gated in practice by the required SSH key. Noted, not overclaimed.

## Acceptance (met)
- [x] Door e2e + full member journey tests pass; menus verified working.
- [x] `sysopctl` creates the first SysOp; panel reachable.
- [x] Graceful shutdown on signal.
- [x] `go build`/`vet`/`test` green (native + linux cross-build); telnet fuzz clean; govulncheck clean.
