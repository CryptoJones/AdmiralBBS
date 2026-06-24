# Sprint 012 — Multi-user Correctness & Door Models | requirements & acceptance

## Why
CryptoJones probed the concurrency story and found more I'd missed: one user
could log in unlimited times at once (multiplying their daily time budget); door
games were modeled as single-player-only (every caller Node 1, jail wiped on
exit — no persistence, no multiplayer, no MajorMUD-style resident games); and the
message boards lacked search / sort / user-filter.

## Fixed / added
- **Per-user session cap** (`session.Presence`, default 1) — a user can't open
  N concurrent sessions to multiply their daily budget.
- **Node pool** (`session.NodePool`) — every session gets a unique node number;
  bounds total concurrent members; fixes "everyone is Node 1".
- **Door models — all three now:**
  - single-player (subprocess, per-node working dir);
  - turn-based file-shared multiplayer (`$DOORSHARE` shared dir);
  - **resident real-time multiplayer (MajorMUD-style)**: a `resident` door kind
    (`store` migration 003 + `doors.Bridge`) that relays each caller to one
    persistent running game server so all share the world. Registrable from the
    SysOp panel.
  - Door working dirs are now **persistent** (state survives plays), not wiped.
- **Message boards: search / sort-by-date / filter-by-user** — `Messages.Search`
  (decrypts + scans, since subject/body are ciphertext), `ThreadSorted`,
  `ByAuthor`; wired into the board menu.

## Honest remaining gaps (flagged, not yet built)
- Pagination across boards/mail/files (long lists dump everything).
- Message/mail/file delete & edit; "new since last visit" read pointers; who's-online.

## Acceptance (met)
- [x] Presence caps concurrent logins (race-tested); node pool unique+bounded.
- [x] Resident-door bridge relays both ways (loopback test); door state persists
      across plays + shared dir works.
- [x] Board search/sort/filter correct (incl. search over encrypted content).
- [x] `go build`/`vet`/`test` green under `-race`; linux cross-build clean; govulncheck clean.
