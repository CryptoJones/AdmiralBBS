# Backlog

This file and the GitHub **[Issues tab](https://github.com/CryptoJones/AdmiralBBS/issues)** are two views of the same list
and must stay in sync. Every backlog item below has a matching GitHub issue and vice
versa — when an item ships and its issue closes, check the box (or move it to `Done`)
here so neither side drifts.

## Open

<!-- Add open items here, each with a matching GitHub issue. -->

- [x] **Switchable colorblind-friendly palette (Claude Code's), light + dark** ([#9](https://github.com/CryptoJones/AdmiralBBS/issues/9)) — accessibility; matches C³ #38. Route `screen` colors through a swappable theme, persist per user. — _shipped v2.2.0_

- [x] **File menu: SysOp creates a new File Area (topic)** ([#6](https://github.com/CryptoJones/AdmiralBBS/issues/6)) — SysOp-gated `[N]ew area` on the File menu, mirroring the message-board `[N]ew board` from #2. — _shipped v2.1.0: SysOp [N]ew File Area_
- [x] **Mail: user-search in `To:` (on `*`) + a new `CC:` field** ([#7](https://github.com/CryptoJones/AdmiralBBS/issues/7)) — typing `*` opens the member-directory search in `To:`; add a `CC:` field with the same lookup. — _shipped v2.1.0: To/CC * search + CC field_
- [x] **Modern file upload** ([#8](https://github.com/CryptoJones/AdmiralBBS/issues/8)) — X/Z-modem doesn't work over SSH; add base64 paste for binaries and/or an HTTP(S) upload endpoint. — _shipped v2.1.0: [B]ase64 binary upload over SSH_

## Done

- [x] **Move SysOp Control Panel off [X] — reserve X and Q for quitting menus** ([#5](https://github.com/CryptoJones/AdmiralBBS/issues/5)) — v2.0.7. SysOp panel is now `[S]`; `X`/`Q` quit/back everywhere.
- [x] **Mail: look up users from the To: prompt** ([#4](https://github.com/CryptoJones/AdmiralBBS/issues/4)) — v2.0.6. `?` opens a paged member directory.
- [x] **Add points to a user's stats** ([#3](https://github.com/CryptoJones/AdmiralBBS/issues/3)) — v2.0.6. `points` column, SysOp `[A]ward points`, profile stats block.
- [x] **Message board: SysOp creates a new board category** ([#2](https://github.com/CryptoJones/AdmiralBBS/issues/2)) — v2.0.6. Password-gated `[N]ew board`.
- [x] **SysOp-configurable session timeouts** ([#1](https://github.com/CryptoJones/AdmiralBBS/issues/1)) — v2.0.4. Idle + daily budget editable from the SysOp panel.

_Earlier work (the four founding features, moderation & abuse, and QoL polish —
sprints S002–S021) predates issue tracking and is recorded in
[`planning/STATE.md`](planning/STATE.md) and [`CHANGELOG.md`](CHANGELOG.md)._

## Decided non-goals (do not build)

- **Password rotation policy** — discredited practice (NIST dropped it); none
  planned. Self-service change exists; max length is the 4096-byte input cap.
- **Blocking IP/VPN roaming** — legitimate VPN users would be punished; roaming
  is allowed by design (the visibility-only rapid-IP-change flag shipped instead).

---

*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
