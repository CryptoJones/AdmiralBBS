# Sprint 004 — Message Boards | requirements & acceptance

## Goal
Members can read and post in message boards, with threaded replies and
per-area access gating. (Implemented in the `implement all the bbs features`
loop, 2026-06-23.)

## Delivered
- `store.MessageAreas` (Create / Visible-by-access-level / Count / ByID) and
  `store.Messages` (Post / Thread / Replies / ByID); message **subject and body
  encrypted at rest** (🔒).
- `EnsureSeedAreas` seeds "General" + "Retro Computing" on first run.
- `menu.RunBoards`: list areas → browse a board → read a message + its replies →
  post / reply (multi-line, end with `.`). All user content rendered via
  `SafePrint` (output escape-sanitisation, SEC-5). Access gating hides boards
  above the member's level.
- Wired into the SSH member menu (`M`).

## Acceptance (met)
- [x] Post + threaded reply round-trip; reply is not a top-level message.
- [x] Restricted area hidden from lower access levels.
- [x] Message body is ciphertext at rest (verified).
- [x] `go build` / `vet` / `test` green.
