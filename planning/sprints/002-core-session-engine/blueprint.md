# Sprint 002 — Core Session Engine | blueprint

File-by-file build plan. Follow `docs/ARCHITECTURE.md` for the layered shape.

## Files to inspect first

- `planning/DECISIONS.md` — stack, transport, hardening (non-negotiable).
- `docs/ARCHITECTURE.md` — the `Conn` / `Session` / menu contracts.
- `docs/VALIDATION.md` — the exit gate and the hardening negative tests.
- ANSI-BBS spec — control-sequence handling + legal-sequence rule.

## Files to create

| Path | Purpose |
|---|---|
| `go.mod`, `go.sum` | Go module (`module admiralbbs`); dep: `golang.org/x/crypto/ssh` |
| `src/cmd/admiralbbs/main.go` | Entry point: load config, start both listeners, drop privileges / refuse root |
| `src/transport/conn.go` | `Conn` interface: `io.ReadWriter` + `WindowSize()` + `TermType()` + window-change channel |
| `src/transport/telnet.go` | Telnet listener `:2323`, IAC negotiation, NAWS parsing |
| `src/transport/ssh.go` | SSH listener `:2222`, host-key bootstrap, pty-req + window-change |
| `src/session/session.go` | `Session` wrapping a `Conn`: lifecycle, time-budget hook |
| `src/session/input.go` | **Hardened reader**: bounded reads, control-char filter, escape-seq validation |
| `src/session/terminal.go` | Capability detection: ANSI vs BW, cols/rows, CP437 |
| `src/screen/writer.go` | Capability-aware writer: ANSI codes when able, plain B&W fallback (one path) |
| `src/screen/ansiart.go` | CP437 `.ans` loader/streamer |
| `src/menu/menu.go` | Menu definition type + dispatcher |
| `src/menu/demo.go` | The static demo menu (welcome / placeholder / logoff) |
| `art/welcome.ans` | A welcome screen (sample or hand-made) |
| `tests/...` | Unit, integration (ephemeral-port scripted client), fuzz (`testing.F`) |

## Step-by-step

1. **Module skeleton** — `go mod init admiralbbs`; create the package dirs;
   `main.go` that refuses to run as root and logs the two bind addresses.
2. **`Conn` contract** — define the interface first; both listeners target it.
3. **Telnet listener** — accept loop → per-conn goroutine → IAC negotiation →
   NAWS → satisfy `Conn`. Validate raw with `nc`/SyncTERM.
4. **SSH listener** — `x/crypto/ssh` server config, generate+persist host key,
   accept `session` channel + `pty-req`/`window-change` → satisfy `Conn`.
5. **Hardened input reader** — bounded buffer, filter control chars to the
   ANSI-BBS safe set, validate escape sequences (legal = `ESC` + `0x40–0x5f`),
   silently drop illegal ones. Write the fuzz test alongside.
6. **Terminal capability** — derive ANSI vs BW from term type + negotiation;
   default 80 cols, never assume rows; mark CP437.
7. **Screen writer** — single render path with a capability flag; ANSI emits
   colour/cursor, BW emits stripped plain text. Load and stream `welcome.ans`.
8. **Menu engine** — data-defined menu + keypress dispatch; wire the demo menu
   (welcome → placeholder → logoff).
9. **Wire main** — both listeners → `Session` → demo menu.
10. **Validate** — run unit/integration/fuzz; connect via SyncTERM on telnet
    AND ssh; capture an ANSI screenshot and a B&W capture into this folder.

## Notes for the Builder

- Goroutine-per-caller; clean up on disconnect (`defer sess.Close()`).
- Keep transport quirks **inside** `src/transport/`; the session/menu layers
  must not know which transport they're on.
- Hardening is not a later sprint — the bounded reader and escape validation
  ship **here**, with their negative tests.
