# Sprint 006 — File Library | requirements & acceptance

## Goal
Members can browse download areas, upload, and download files, with sealed-at-rest
blobs, access gating, and SEC-7 hardening. (Built 2026-06-23.)

## Delivered
- `store.FileAreas` (Create / Visible-by-access / Count / ByID) and `store.Files`
  (Add / ListByArea / ByID / Content). Blobs are **sealed with the vault and
  written to disk by row id** (`<id>.bin`) — never from the caller's filename, so
  path traversal is structurally impossible (SEC-7). Size capped at 10 MiB.
- `Store` gained a `filesDir` (`<db-dir>/files`, 0700, gitignored).
- `EnsureSeedFileAreas` seeds "General Files" on first run.
- `menu.RunFiles`: browse areas → list files (name/size/downloads/desc) →
  download (decrypts, streams between markers, bumps counter) / upload (text/ANSI
  via paste; binary X/Y/Zmodem is a planned follow-on). All user content via
  `SafePrint` (SEC-5). Wired into the member menu (`F`).

## Acceptance (met)
- [x] Upload → list → download round-trips; download_count increments.
- [x] Oversize upload rejected (ErrTooLarge).
- [x] Hostile filename (`../../../../etc/passwd`) cannot escape the files dir.
- [x] Blob is ciphertext at rest (verified). `go build`/`vet`/`test` green.

## Follow-on
- Binary X/Y/Zmodem transfer; per-user upload quotas; SysOp file-area management (S008).
