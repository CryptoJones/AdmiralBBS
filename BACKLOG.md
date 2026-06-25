# Backlog

This file and the GitHub **[Issues tab](https://github.com/CryptoJones/AdmiralBBS/issues)** are two views of the same list
and must stay in sync. Every backlog item below has a matching GitHub issue and vice
versa — when an item ships and its issue closes, check the box (or move it to `Done`)
here so neither side drifts.

## Open

<!-- Add open items here, each with a matching GitHub issue. -->

_Nothing open — every tracked issue has shipped._

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
