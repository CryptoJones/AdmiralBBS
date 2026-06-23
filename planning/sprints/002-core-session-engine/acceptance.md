# Sprint 002 — Core Session Engine | acceptance

A sprint is **not** done because the code compiles. It is done when every
criterion below is objectively true.

## Functional criteria

- [ ] `go build ./...` produces a single binary; `go test ./...` is green.
- [ ] The binary **refuses to run as root** and logs both bind addresses.
- [ ] A caller can connect over **Telnet** (`:2323`) with SyncTERM/NetRunner
      and see the ANSI welcome screen render correctly (colour + CP437 box art).
- [ ] A caller can connect over **SSH** (`:2222`) and see the same screen.
- [ ] The **same screen is readable in plain B&W** on a non-ANSI terminal
      (degraded, not garbled).
- [ ] The demo menu responds to keypresses and a logoff option cleanly
      disconnects the caller.
- [ ] Window size is detected (Telnet NAWS and SSH window-change) and a resize
      is reflected.

## Hardening criteria (negative tests — `docs/VALIDATION.md`)

- [ ] Fuzzing the input reader (`testing.F`) with oversized / binary /
      malformed streams never crashes the process and never grows memory
      unbounded.
- [ ] Hostile control chars and illegal escape sequences are filtered/ignored
      per the ANSI-BBS legal-sequence rule; they never reach the terminal raw.
- [ ] Transport quirks are contained in `src/transport/`; `session`/`menu`
      packages have no transport-specific code (verified by inspection/imports).

## Evidence to capture in this folder

- [ ] A SyncTERM screenshot of the ANSI welcome (telnet and/or ssh).
- [ ] A capture showing the B&W degraded render.

## Process criteria

- [ ] `planning/STATE.md` updated to reflect Sprint 002 status / next action.
- [ ] Any new durable choice (e.g. hashing lib, config format) recorded in
      `planning/DECISIONS.md`.
- [ ] Any ambiguity hit during the sprint was added to `planning/QUESTIONS.md`,
      not silently guessed.
