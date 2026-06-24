# Sprint 008 — SysOp Control Panel | requirements & acceptance

## Goal
The operator's in-BBS admin console — SSH-only, access ≥80, server-side gated on
every action. (Built 2026-06-23.)

## Delivered
- `menu.RunSysOp` (shown on the main menu as `X` only to access ≥80):
  - **Membership queue** — list pending applications + notes; approve (set level
    + issue the one-time onboarding token, shown once, to relay out-of-band) or deny.
  - **User management** — list users; set access level, suspend/reinstate, set
    daily-minutes, clear password (force re-onboarding). Never shows hashes.
  - **Content management** — create message areas, file areas, register doors.
  - **Audit viewer** — recent `session_log` events (detail decrypted) **and**
    end-to-end verification of the authoritative JSONL hash-chain (SEC-6),
    surfacing tampering.
- Store helpers: `Users.All` / `SetDailyMinutes`, `SessionLog.Recent`,
  `Store.VerifyAuditChain`.

## Acceptance (met)
- [x] Panel only offered to access ≥80 and re-checks server-side.
- [x] Approve sets status/level and issues a token that redeems for onboarding.
- [x] User management (daily-minutes, suspend) persists.
- [x] Audit viewer decrypts detail and the chain verifies intact.
- [x] `go build`/`vet`/`test` green.
