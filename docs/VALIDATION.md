# Validation Plan — AdmiralBBS

> Validation is against **real BBS behaviour and the hardening mandate**, not
> just green unit tests (per `AGENTS.md`). A feature is "done" when a real
> terminal client behaves correctly AND the relevant attack surface is proven
> closed.

## How trust is proven

### 1. It works with a real BBS terminal
The ground truth for a 90s BBS is a period-correct client, not `telnet` or a
unit test. Every interactive feature must be validated by connecting with
**SyncTERM** and/or **NetRunner** over both transports:

- ANSI screens render correctly (colour, CP437 box-drawing, cursor placement).
- The **same screens degrade to readable B&W** on a non-ANSI / plain terminal.
- Window resize is handled (SSH `window-change`, Telnet NAWS).

### 2. The hardening mandate is adversarially tested
Each item in the `DECISIONS.md` hardening list gets an explicit negative test:

| Mandate | Validation |
|---|---|
| Buffer overflow | Fuzz the input reader with oversized / binary / malformed streams; the session must bound, reject, or sanitise — never crash or grow memory unbounded. |
| Packet injection | Feed hostile escape sequences and control chars; assert they are filtered/ignored per the ANSI-BBS legal-sequence rule and never reach a parser or the terminal raw. |
| Sandbox escape | Launch a door that *attempts* to read parent-FS paths / escalate; assert the chroot/uid jail denies it. |
| Non-root posture | Assert the daemon refuses to keep root and binds only its high ports. |
| Credentials | Assert no plaintext password is ever written to the DB or logs. |

### 3. Automated test layers
- **Unit** — pure logic (input sanitiser, ANSI/B&W writer, menu dispatch, store repos).
- **Integration** — spin a listener on an ephemeral port, drive it with a scripted in-process client, assert the byte stream.
- **Fuzz** — Go's native `testing.F` against the input reader and protocol negotiators.

## Per-sprint exit gate

A sprint does not close until:
1. Its `acceptance.md` criteria are each objectively true.
2. A real SyncTERM/NetRunner session demonstrates the feature (screenshot or capture in the sprint folder) where the feature is interactive.
3. Any attack surface the sprint introduced has a passing negative test.
4. `planning/STATE.md` is updated.

## What "AI output is not source of truth" means here

Generated numbers, generated ANSI art, and generated test assertions are
suspect until a human or a real client confirms them. Door-game behaviour and
terminal rendering especially must be eyeballed in a real client, not trusted
from a passing test alone.

---

*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
