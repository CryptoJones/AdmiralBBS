# Sprint 007 — Door Games | requirements & acceptance

## Goal
Members can launch door games that run as SANDBOXED subprocesses wired to their
session — realising the founding "AI/door escapes to the parent OS" risk
mitigation (SEC-1). (Built 2026-06-23.)

## Delivered
- `src/doors` package — `Launch()` runs a door as a subprocess with:
  - **fully scrubbed environment** (built from scratch: PATH, HOME=jail, TERM,
    DOORFILE) so `ADMIRALBBS_KEY` and every secret are invisible to the door;
  - a **throwaway jail working dir** (0700) holding only the `door32.sys` dropfile;
  - a **CPU rlimit** (via `ulimit` in the child) + a **wall-clock timeout**;
  - its **own process group**, SIGKILL'd as a group on timeout/disconnect so the
    door and any children die.
  - `WriteDoor32` generates the standard 11-line door32.sys dropfile.
- `store.Doors` repo (Create / Visible-by-access / ByID / Count) + `EnsureSeedDoors`
  (registers the bundled demo door if present).
- `session.Raw()` — raw byte ReadWriter over the connection (resets the idle
  watchdog) so the door pipes directly to the caller.
- `menu.RunDoors` — list + launch; wired into the member menu (`D`).
- Bundled original demo door `doors/numguess.sh` (reads the dropfile, plays a
  number-guess round over stdio).

## Acceptance (met)
- [x] A door CANNOT see `ADMIRALBBS_KEY` or any inherited env (verified).
- [x] A runaway door is killed at the wall-clock timeout (its whole group).
- [x] door32.sys generated with correct fields; doors repo access-gated.
- [x] Bundled demo door reads the dropfile + plays. `go build`/`vet`/`test` green.

## Follow-on (S009 deploy hardening)
- Run door execution under a dedicated unprivileged uid / container (chroot,
  namespaces, seccomp) — needs privilege/containerisation, layered beneath this.
- Binary/legacy DOS door interop (FOSSIL) if ever wanted.
