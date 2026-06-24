# Sprint 010 — Feature Polish | requirements & acceptance

## Goal
The four follow-on features noted at the end of S009. (Built 2026-06-23.)

## Delivered
1. **Per-user upload quotas** — `store.MaxUserBytes` (100 MiB) + `Files.UserBytes`;
   `Files.Add` rejects over-quota with `ErrQuotaExceeded` (surfaced in the menu).
2. **Key rotation** — `cmd/rekey` re-encrypts everything from the OLD key to a NEW
   key: `store.RekeyDB` re-seals all sealed DB columns + file-library blobs;
   `audit.Rekey` re-seals + re-chains the audit JSONL; the new KDF salt is written
   only on full success (mid-run failure leaves the old key recoverable). Secrets
   come from env only.
3. **Door uid/namespace isolation** — `doors.Opts` gains `RunAsUID/GID` (portable
   Credential drop), `Chroot`, `NoNetwork`, `Isolate`; Linux build (`isolate_linux.go`)
   adds chroot + fresh mount/pid/ipc/uts namespaces, an **empty network namespace
   (no network for the door)**, and `Pdeathsig`; no-op off Linux. Wired through
   `-door-uid/-door-gid/-door-chroot/-door-no-network/-door-isolate` flags. (Opt-in;
   needs privilege — layers beneath the always-on in-process sandbox.)
4. **XMODEM binary transfer** — `src/xfer` (CRC-16/XMODEM `Send`/`Receive`); the
   file menu offers XMODEM download/upload (vs. inline view / text paste).

## Acceptance (met)
- [x] Over-quota upload rejected; under-quota allowed.
- [x] Full rekey round-trip: DB fields, file blob, and audit all readable under
      the NEW key and NOT the old; audit chain re-verifies.
- [x] Isolation opts compile + apply (Linux cross-build verified); default
      non-root run unaffected; door env still scrubbed.
- [x] XMODEM Send/Receive interoperate across block boundaries (1B–4KB).
- [x] `go build`/`vet`/`test` green (native + linux cross-build); govulncheck clean.
