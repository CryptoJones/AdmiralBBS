# BACKLOG — what's left

_Last updated: 2026-06-24. Single source of truth for unbuilt work. As items
ship, move them to STATE.md's "Recently completed" and delete them here._

The four founding features (boards, mail, files, doors), the SysOp console,
2FA-SSH auth, encryption in transit + at rest, and CI (green @ `e9d0dfd`) are
done. What remains is moderation/abuse tooling and quality-of-life polish.

## P1 — Moderation & abuse (the real gap)

These are the items CJ surfaced that are genuinely missing. Suggested order is
cheapest-anti-abuse-win first.

- [ ] **SSH-key fingerprint uniqueness** — one account per public-key
  fingerprint. Today the same key can register on multiple accounts, so
  key-pairs don't stop sockpuppets. Add a uniqueness constraint on the key
  fingerprint (race-safe via DB constraint, like the handle) + a clear error on
  collision. Cheapest anti-sockpuppet win.
- [ ] **SysOp IP banlist + transport enforcement** — SysOp can ban an IP /
  CIDR; the Telnet and SSH listeners reject banned sources at connect time
  (before auth). Needs a `banlist` table, a SysOp panel entry, and an
  enforcement hook in both transports.
- [ ] **User-to-user moderation** — block/ignore (a user mutes another so their
  messages/mail are hidden) + report-to-SysOp (a report lands in the SysOp
  queue for review/action).

## P2 — Quality-of-life polish (flagged but not blocking)

- [ ] **Pagination** — boards, file lists, and mail currently dump full lists;
  add paging for large areas.
- [ ] **Message / mail edit + delete** — authors can edit or delete their own
  posts/messages (SysOp can delete any).
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
