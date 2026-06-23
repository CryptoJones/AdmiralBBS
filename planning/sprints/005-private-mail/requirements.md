# Sprint 005 — Private Mail | requirements & acceptance

## Goal
User-to-user private mail: compose, inbox with unread tracking, read, reply.
(Implemented in the `implement all the bbs features` loop, 2026-06-23.)

## Delivered
- `store.Mail`: Send / Inbox (newest first) / UnreadCount / Get (marks read for
  the recipient, enforces party-only access). Subject + body **encrypted at
  rest** (🔒).
- `menu.RunMail`: inbox with `[NEW]` markers, read (auto-marks read), compose
  (recipient by handle), reply. User content via `SafePrint` (SEC-5).
- Wired into the SSH member menu (`E`).

## Acceptance (met)
- [x] Send → recipient inbox shows it unread; UnreadCount = 1.
- [x] Reading as recipient marks read; UnreadCount → 0; body decrypts.
- [x] A third party gets ErrNotFound (party-only).
- [x] Body ciphertext at rest (verified). `go build`/`vet`/`test` green.
