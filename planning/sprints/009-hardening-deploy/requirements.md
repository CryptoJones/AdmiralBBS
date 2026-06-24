# Sprint 009 — Hardening & Deploy | requirements & acceptance

## Goal
Final hardening + deployment polish; close the SEC-10/11/12 deploy items. (Built 2026-06-23.)

## Delivered
- **CI** (`.github/workflows/ci.yml`, GitHub-canonical): build + vet + test +
  `govulncheck` on every push/PR. `govulncheck ./...` is clean (no vulns).
- **Container hardening** (`docker-compose.yml`, SEC-10): read-only root FS,
  `tmpfs /tmp` for door jails, `cap_drop: ALL`, `no-new-privileges`, non-root
  user, master key injected from host env (required), persistent state volume.
- **`docs/OPERATIONS.md`**: native + container run, the honest threat model, and
  a **key-rotation runbook** (SEC-12).

## Acceptance (met)
- [x] `go build`/`vet`/`test` green; `govulncheck` reports no vulnerabilities.
- [x] Compose runs hardened (read-only, dropped caps, no-new-privileges, non-root).
- [x] Operations + key-rotation runbook documented.

## Noted as future (need privilege/scale, not blockers)
- Run door execution under a dedicated uid / full container isolation (seccomp
  profile, namespaces) layered beneath the in-process sandbox.
- `scripts/rekey` to make key rotation an in-place re-encrypt.
- Per-user upload quotas; binary X/Y/Zmodem file transfer.
