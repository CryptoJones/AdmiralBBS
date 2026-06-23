# Architecture — AdmiralBBS

> Clean-room implementation of a 90's-era ANSI BBS, in Go. This doc is the
> stable map of how the pieces fit. Decisions live in `planning/DECISIONS.md`;
> this file explains the shape they produce.

## One-paragraph overview

AdmiralBBS is a single static Go binary that listens on two transports
(Telnet and SSH), wraps every caller in a transport-agnostic **Session**,
detects the caller's terminal capability (ANSI colour vs. plain B&W), and
drives them through a **menu/screen engine**. Subsystems (message boards,
private mail, file library, door games) are modules the menu engine routes
into. All state persists to an embedded **SQLite** file. Door games run as
**sandboxed OS subprocesses**, never in-process.

## Layered view

```text
            ┌──────────────────────────────────────────────┐
 callers →  │  Transport layer                             │
            │   net_telnet (:2323)   net_ssh (:2222)       │
            │   - IAC negotiation     - x/crypto/ssh        │
            │   - NAWS (window size)  - pty/window-change   │
            └───────────────┬──────────────────────────────┘
                            │  yields a raw io.ReadWriter + window size
            ┌───────────────▼──────────────────────────────┐
            │  Session  (transport-agnostic)               │
            │   - bounded, validated input reader (HARDEN) │
            │   - terminal capability (ANSI|BW, cols/rows) │
            │   - per-caller state, time budget            │
            └───────────────┬──────────────────────────────┘
            ┌───────────────▼──────────────────────────────┐
            │  Screen/Menu engine                          │
            │   - ANSI writer w/ graceful B&W degrade      │
            │   - CP437 art rendering, .ans file loader    │
            │   - menu definitions + dispatch              │
            └───┬──────────┬──────────┬──────────┬─────────┘
                │          │          │          │
          ┌─────▼───┐ ┌────▼────┐ ┌───▼────┐ ┌───▼─────────┐
          │ Message │ │ Private │ │ File   │ │ Door games  │
          │ boards  │ │ mail    │ │ library│ │ (sandboxed  │
          │         │ │         │ │        │ │  subprocess)│
          └─────┬───┘ └────┬────┘ └───┬────┘ └───┬─────────┘
                └──────────┴──────────┴──────────┘
                            │
            ┌───────────────▼──────────────────────────────┐
            │  Store  (embedded SQLite, single file)       │
            │   users, memberships, messages, files, ...   │
            └──────────────────────────────────────────────┘
```

## Components

### Transport layer (`src/transport/`)
Two listeners, one contract. Each accepts a connection and hands the session
engine a `Conn` interface: `io.ReadWriter` + a window-size channel + the
negotiated terminal type string.

- **Telnet** — handle IAC negotiation (suppress-go-ahead, echo, binary), parse
  **NAWS** for window dimensions, strip/handle the control sequences the
  ANSI-BBS spec flags as dangerous. Plaintext by design (authenticity).
- **SSH** — `golang.org/x/crypto/ssh`, host key on first run, `pty-req` and
  `window-change` give us terminal type + live resize for free.

Both are insecure to expose naively, so the daemon binds high ports
(`:2323`, `:2222`) and runs as a **non-root** user.

### Session (`src/session/`)
The unit everything else is written against. Owns:
- The **hardened input reader**: every read is length-bounded, control chars
  are filtered to a known-safe set, escape sequences validated against the
  ANSI-BBS legal-sequence rule before they ever reach a parser. This is where
  the "buffer overflow / packet injection" mandate is enforced.
- **Terminal capability**: ANSI vs B&W, columns (assume 80 if unknown), rows
  (never assumed), CP437.
- **Time budget** (the per-day minute limit — see `planning/QUESTIONS.md`).

### Screen / Menu engine (`src/screen/`, `src/menu/`)
- ANSI writer that emits colour + cursor codes when the session is ANSI-capable
  and **silently degrades to plain text** otherwise (single code path, capability flag).
- CP437 `.ans` art loader for login screens / menus.
- Menus are data (definitions) + a dispatcher that routes a keypress to a
  subsystem handler.

### Subsystems (`src/boards/`, `src/mail/`, `src/files/`, `src/doors/`)
Each is a self-contained module the menu engine routes into, talking to the
Store. **Doors** are special: they launch an external program as a sandboxed
subprocess (dedicated uid, chroot/jail, no parent-FS access), write the
classic dropfile (`door32.sys` / `DOOR.SYS`) for the door to read, and pipe
the caller's session I/O to it.

### Store (`src/store/`)
Embedded SQLite behind a repository interface, so subsystems never touch SQL
directly and the backend stays swappable. See `docs/DATA_MODEL.md`.

## Cross-cutting: the hardening posture

| Threat (from RISKS / DECISIONS) | Mitigation |
|---|---|
| Buffer overflow | Memory-safe Go; no manual buffer math |
| Packet injection / malformed input | Bounded reader, validated escape sequences, filtered control chars at the Session boundary |
| Sandbox escape (via doors / "AI that escapes to parent OS") | Doors run as a separate uid in a chroot/jail subprocess; the BBS daemon itself runs non-root with least privilege |
| Plaintext credentials | SSH transport available; passwords hashed at rest (never stored plain) |

## Deployment

`go build` → one static binary + one SQLite file + an `art/` directory of
`.ans` screens. Run as a non-root service user. No external services required.

---

*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
