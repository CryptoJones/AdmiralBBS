# RISKS — known traps

A living register. Each risk notes **severity**, the **mitigation**, and **when**
it lands (foundation = baked into the core design now; Sxxx = the sprint that
owns it; deploy = operational/container concern). When a mitigation is fully
realised, also record it as a decision in `DECISIONS.md`.

## Founding risks (from the operator)

- **Mythos-level AI / hostile callers that log in and escape into the parent OS.**
  _Severity: critical._ Mitigation: memory-safe Go (no buffer overflows), all
  caller input sanitised at the session boundary, and door games run as
  sandboxed subprocesses (see SEC-1). Foundation + S007.
- **Terminal-emulation attack surface.** Hostile IAC/ANSI/escape sequences.
  _Severity: high._ Mitigation: hardened input reader + ANSI-BBS legal-sequence
  validation (shipped S002); fuzz the telnet/SSH protocol parsers, not just the
  sanitiser (SEC-6). Foundation.

## Security register (100,000 ft review, 2026-06-23)

### 🔴 Baked into the foundation now

- **SEC-1 — Door games are the #1 attack surface, and the master key must not
  leak into them.** _Critical._ Doors run arbitrary programs. Isolation:
  separate uid + chroot/jail, **scrubbed environment (strip `ADMIRALBBS_KEY` and
  all secrets before exec)**, dropped capabilities, rlimits (CPU/mem/procs/file
  size), wall-clock timeout, no inherited fds, **no network**. Foundation
  (design) + S007 (implementation).
- **SEC-2 — Account-takeover window in the "set password on first SSH login"
  flow.** _High._ Between SysOp approval and the applicant's first SSH login,
  anyone SSHing with that handle can claim the password. Mitigation: a
  **one-time approval token** delivered out-of-band (or capture the applicant's
  SSH public key at application time), required to set the password.
  Foundation (design) + S003.
- **SEC-3 — DoS / resource exhaustion.** _High._ Goroutine-per-connection with
  no cap is trivially exhausted. Mitigation: max concurrent sessions, per-IP
  connection throttle, SSH handshake timeout, idle timeout, and slow-loris read
  deadlines. Foundation (transport/session layer).
- **SEC-4 — Login brute-force & user enumeration.** _High._ Mitigation: rate
  limit + exponential backoff/lockout on BBS password auth; a **generic
  "login failed"** message; constant-time user lookup so timing doesn't leak
  which handles exist. Foundation (design) + S003.
- **SEC-5 — Stored-content ANSI/escape injection (user → user).** _High._ We
  sanitise input, but message bodies and handles rendered to *other* callers
  must be escape-sanitised on **output** too, or one user hijacks another's
  terminal. Foundation (screen layer) + S004.

### 🟠 Audit-trail integrity (correction to an earlier claim)

- **SEC-6 — Encryption ≠ tamper-evidence.** _Medium-high._ Sealing the audit
  log gives confidentiality, not integrity; with the key or volume write access
  an attacker could alter/delete log lines undetectably. Mitigation: an
  **HMAC hash-chain** (each line commits to the previous) plus append-only
  semantics, so any edit/truncation is detectable. Decision: adopt
  confidentiality **+ hash-chain integrity**. Foundation (audit layer).

### 🟡 Per-sprint, when the feature lands

- **SEC-7 — File library path traversal & abuse.** _High._ `../` in filenames,
  upload size/quota limits, zip-bomb defense; file blobs sealed at rest;
  uploads never executable. S006.
- **SEC-8 — Server-side authorization everywhere.** _High._ Access-level checks
  enforced on every action, not just by hiding menu items. First-SysOp
  bootstrap must not be a hardcoded default password. S003 + every later sprint.
- **SEC-9 — Telnet leaks application PII.** _Medium._ Even without a password,
  the application screen collects email/real-name in plaintext (sniffable).
  Minimise what is gathered over Telnet. S003.

### ⚪ Deployment / supply chain

- **SEC-10 — Container hardening.** _Medium._ Read-only root FS, `no-new-privileges`,
  dropped Linux capabilities, seccomp profile, only 2323/2222 exposed. Prefer
  **Docker secrets (file-mounted)** over env vars for the key (env leaks via
  `/proc/<pid>/environ` and child inheritance). Deploy.
- **SEC-11 — Supply-chain / dependency CVEs.** _Medium._ Add `govulncheck` to
  CI, pin dependencies (`go.sum` present), use a minimal base image, scan
  images. GitHub push secret-scanning already on. Deploy/CI.
- **SEC-12 — Weak secret & no key rotation.** _Medium._ The Argon2id KDF only
  protects a weak `ADMIRALBBS_KEY` so far; document a strong-secret requirement
  and a re-encrypt/rotation path for a compromised key. Deploy + future sprint.

### Honest threat-model boundary

Encryption at rest is **fully effective against offline access** — stolen disk,
copied volume, image layer, backup, or stopped container. It **cannot fully
defend against an attacker with live root** on the running host, who can scrape
the in-memory key; `mlock` + zeroing raise the bar but do not eliminate it.
Closing that gap requires hardware (TPM/HSM/enclave), out of scope for this
Go + SQLite + container stack. Do not claim otherwise in docs or to users.

## Always-on risks for any 120x project

- AI output is not source of truth. Numbers must trace back to data, documents, or human confirmation.
- Single-file overload — context must be split across the planning files, not crammed into one.
- Tool churn — the methodology must survive any specific agent going away.
