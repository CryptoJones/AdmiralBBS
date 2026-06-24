# BACKLOG — what's left

_Last updated: 2026-06-24. Single source of truth for unbuilt work. As items
ship, move them to STATE.md's "Recently completed" and delete them here._

The four founding features (boards, mail, files, doors), the SysOp console,
2FA-SSH auth, encryption in transit + at rest, and CI (green @ `e9d0dfd`) are
done. What remains is moderation/abuse tooling and quality-of-life polish.

## P1 — Moderation & abuse — COMPLETE (S014–S016)

All three landed on `main`. The moderation gap CJ surfaced is closed.

- [x] **SSH-key fingerprint uniqueness** — DONE (S014, `main`). One account per
  public-key fingerprint, enforced by a partial unique index on active keys
  (migration 004) so it's race-safe like the handle constraint. `ErrKeyTaken`
  surfaced in the apply flow and profile add; `Keys.ByFingerprint` resolves a
  key to its single owner. Revoking frees the fingerprint for re-use.
- [x] **SysOp IP banlist + transport enforcement** — DONE (S015, `main`).
  `ip_ban` table (migration 005, exact IP or CIDR, soft-lift), `store.Bans`
  repo (Add/Lift/Active/IsBanned, fail-open on error), SysOp panel `[B]`
  add/lift/list, and a `BanCheck` hook that drops banned sources at accept time
  in BOTH transports before auth. End-to-end test proves the Telnet listener
  rejects a banned source (handler never runs).
- [x] **User-to-user moderation** — DONE (S016, `main`). `user_block` (personal,
  one-directional mute) + `report` queue (migration 006). `store.Blocks`
  (Block/Unblock/IsBlocked/BlockedSet/List) and `store.Reports`
  (File/Open/Resolve/OpenCount). Mail inbox and board threads/replies hide
  blocked users; read views offer [B]lock + re[P]ort; profile has block
  management; SysOp panel [R] reviews/resolves reports (with suspend). E2E test
  proves the mail menu actually hides a blocked sender.

## P2 — Quality-of-life polish (flagged but not blocking)

- [x] **Pagination** — DONE (S017, `main`). Shared `menu` pager (pageWindow /
  clampPage / pageFooter, 15 rows/page) wired into the mail inbox, board
  threads, and file browse. `[>]`/`[<]` paging keys (chosen to avoid colliding
  with [P]ost/[B]lock). Unit-tested math + an e2e test that drives the real mail
  menu across two pages.
- [x] **Message / mail edit + delete** — DONE (S018, `main`).
  `Messages.Edit` (owner-only, `ErrNotOwner`) + `Messages.Delete` (post and its
  replies); `Mail.Delete` (recipient-only). Board read view offers [E]dit/[D]elete
  to the author and [D]elete to SysOp; mail read view offers [D]elete. Tested:
  ownership enforcement, cascade delete, and recipient-only mail delete.
- [ ] **"New since last visit" read pointers** — per-user last-read markers so
  the boards show what's unread.
- [ ] **Who's-online** — list currently-connected nodes/users.

## P3 — Visibility / nice-to-have

- [ ] **Impossible-travel anomaly flagging** — surface (do NOT block) logins
  that hop implausibly fast between distant geos, for SysOp awareness. By design
  IP/VPN roaming is allowed, so this is a flag, never an auto-deny.

## Decided NON-goals (do not build)

- **Password rotation policy** — discredited practice (NIST dropped it); none
  planned. Self-service change exists; max length is the 4096-byte input cap.
- **Blocking IP/VPN roaming** — legitimate VPN users would be punished; roaming
  is allowed by design (see P3 flag for the visibility-only alternative).

---
*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
