# Operations — AdmiralBBS

> How to run, secure, and maintain AdmiralBBS. Encryption is mandatory: the
> daemon refuses to start without `ADMIRALBBS_KEY`.

## The master key

- `ADMIRALBBS_KEY` is the startup secret. It is run through Argon2id (with a
  persisted, non-secret salt at `<data>/key.salt`) to derive the in-memory
  XChaCha20-Poly1305 key. The key is `mlock`'d and zeroed on exit, and is
  **never written to the data volume**.
- Supply it via the host environment or a Docker/secret manager — never commit
  it, never bake it into an image, never paste it in chat.
- **Use a strong, high-entropy secret.** The Argon2id KDF only slows an offline
  guess of a weak secret; it is not a substitute for one.

## Run natively

```sh
go build -o admiralbbs ./src/cmd/admiralbbs
ADMIRALBBS_KEY='<strong-secret>' ./admiralbbs \
  -telnet :2323 -ssh :2222 -db data/admiralbbs.db \
  -audit data/audit.jsonl -salt data/key.salt \
  -hostkey data/ssh_host_ed25519_key -art art/welcome.ans
```

Run as an unprivileged user (the daemon refuses to run as root).

## Run in a container (hardened)

```sh
ADMIRALBBS_KEY='<strong-secret>' docker compose up --build
```

The compose file ships read-only root FS, `tmpfs /tmp` (door jails), `cap_drop:
ALL`, `no-new-privileges`, a non-root user, and a persistent volume for state.
For defence against a stolen disk, also place the host volume on an **encrypted
filesystem** (LUKS / encrypted volume) — the app-level encryption protects
content, the encrypted volume covers structural metadata too.

## First SysOp (bootstrap)

The control panel is gated to access level ≥80, and access level is only granted
*by* a SysOp — so a fresh BBS needs a one-time bootstrap, done on the host with
`sysopctl` (it needs `ADMIRALBBS_KEY`):

1. The prospective admin connects over **Telnet** and submits a membership
   application (handle + their SSH public key).
2. On the host:
   ```sh
   ADMIRALBBS_KEY=... sysopctl -db data/admiralbbs.db -salt data/key.salt approve <handle> 100
   ```
   This approves the user at level 100 and prints a **one-time onboarding token**.
3. The admin connects over **SSH** (their key now authenticates), enters the
   token, and sets a password. They now see the **SysOp Control Panel** (`X`).

`sysopctl list` shows all users + status; `promote <handle> [level]` adjusts an
existing user directly.

## Concurrency limits

- `-nodes N` — max simultaneous member sessions (node pool; each caller gets a
  unique node number). `-max-per-user N` — concurrent sessions per user (default
  1, "one node per caller"; prevents multiplying the daily time budget by logging
  in many times at once). Plus the transport `-max-sessions` / `-per-ip` caps.

## Door games (three kinds)

- **Single-player** — `subprocess`; each player gets their own process in a
  per-node working dir (`<doors-data>/<slug>/node<N>/`).
- **Turn-based file-shared multiplayer** (LORD-style) — `subprocess` too; shared
  state goes in `<doors-data>/<slug>/shared/`, passed to the door as `$DOORSHARE`.
  The unique node number namespaces per-player files.
- **Resident real-time multiplayer** (MajorMUD / Worldgroup) — `resident`; the
  game runs as its own persistent server and the BBS BRIDGES each caller to it
  (`net_type`+`address`), so all callers share one live world. Register via the
  SysOp panel (`X` → Create → resident) and run the game server separately.

`-doors-data <dir>` sets the persistent door-data root (default `<db-dir>/doors-data`).

## Threat model (be honest with yourself)

- **Effective against offline access:** a stolen disk, copied volume, image
  layer, backup, or stopped container is ciphertext without the key.
- **NOT a defence against live root** on the running host — root can scrape the
  in-memory key. Closing that needs hardware (TPM/HSM/enclave), out of scope.
- Door games run sandboxed (scrubbed env, jail dir, rlimits, timeout, group
  kill); for stronger isolation run them under a dedicated uid / container.

## Key rotation runbook

The master key encrypts PII, message/mail bodies, file blobs, and the audit
trail. To rotate (e.g., suspected compromise):

1. **Stop** the daemon.
2. **Back up** the data dir.
3. **Re-encrypt** with a one-off migration: open the DB + files with the OLD
   `crypto.Vault`, decrypt every sealed field/blob and the audit JSONL, then
   re-seal with a NEW vault (new `ADMIRALBBS_KEY` + regenerated `key.salt`).
   (A `scripts/rekey` helper is the place for this; it walks `user`,
   `membership`, `message`, `private_message`, `session_log.detail`, and the
   `files/` blobs.)
4. Replace `ADMIRALBBS_KEY` everywhere it is injected and **restart**.
5. Verify: `VerifyAuditChain` passes and a member can log in.

Until `scripts/rekey` exists, treat key rotation as a planned maintenance task,
not an in-place hot swap — a new key cannot read data sealed with the old one.

## Maintenance

- CI (`.github/workflows/ci.yml`) runs build, vet, tests, and `govulncheck` on
  every push/PR. GitHub push secret-scanning is the backstop against an
  accidentally committed credential.
- Audit integrity: the JSONL trail is hash-chained; the SysOp panel verifies it.
  Investigate any "chain verify failed".

---

*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
