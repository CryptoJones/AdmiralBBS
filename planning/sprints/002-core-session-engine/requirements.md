# Sprint 002 — Core Session Engine | requirements

## Goal

Build the **spine** of AdmiralBBS: a caller can connect over **both** Telnet
and SSH, the system detects their terminal (ANSI colour vs. plain B&W), and
they navigate a static, data-driven menu rendered from CP437 `.ans` art. No
user accounts or subsystems yet — those hang off this spine in later sprints.

## User story

As a caller with a retro terminal client (SyncTERM/NetRunner) **or** a plain
terminal, I want to connect to AdmiralBBS and see a correctly-rendered welcome
screen and menu — in full ANSI colour if my client supports it, or readable
black-and-white if it doesn't — so that the board feels like a real 90s BBS
from the first keystroke.

## In scope

- Go module + project skeleton (`go.mod`, `src/` packages, `main`).
- **Telnet listener** (`:2323`): IAC negotiation (suppress-go-ahead, echo,
  binary), NAWS window-size parsing.
- **SSH listener** (`:2222`): `golang.org/x/crypto/ssh`, host key generated on
  first run, `pty-req` + `window-change` handling.
- A transport-agnostic **`Conn`** both listeners satisfy.
- **`Session`**: hardened input reader (length-bounded, control-char filtered,
  escape sequences validated per ANSI-BBS rules), terminal-capability struct
  (ANSI|BW, cols default 80, rows never assumed, CP437).
- **Screen writer**: emits ANSI when capable, **degrades to plain B&W on one
  code path** otherwise. CP437 `.ans` file loader.
- **Menu engine**: data-defined menus + keypress dispatcher; a static demo
  menu (e.g. welcome → [M]essage area placeholder, [G]oodbye/logoff).
- Daemon runs as a **non-root** user; binds only its two high ports.

## Out of scope

- User accounts, login, registration, membership (Sprint 003).
- Message boards, mail, files, doors (Sprints 004–007).
- Persistence / SQLite (Sprint 003 introduces the store).
- Any transfer protocol (Zmodem etc.).

## Inputs

- `planning/DECISIONS.md` (stack, transport, hardening).
- `docs/ARCHITECTURE.md` (layered design, the `Conn`/`Session` contract).
- `docs/VALIDATION.md` (how this sprint is proven done).
- ANSI-BBS spec: <http://ansi-bbs.org/ansi-bbs-core-server.html>
- A few `.ans` welcome screens placed in `art/` (sample or hand-made).

## Outputs

- Buildable Go binary that serves both transports.
- A caller can connect and navigate the demo menu in ANSI and in B&W.
- Unit + integration + fuzz tests per `docs/VALIDATION.md`.
