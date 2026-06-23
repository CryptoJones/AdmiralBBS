# Sprint 003 — Users & Membership (2FA auth) | requirements

## Goal

Turn the data layer into a working membership system: approve applicants, onboard
them, and authenticate every SSH connection with **two factors** — a registered
SSH key (something you have) AND a password (something you know).

## User story

As a SysOp I approve a pending applicant and hand them a one-time token
out-of-band; as that approved member I connect over SSH with my registered key,
set my password on first login, and from then on log in with key + password.

## In scope

- **Approval + one-time token:** SysOp approves a pending user → status becomes
  `approved`, access level set, and a single-use, time-limited token is issued
  (stored hashed). Token plaintext is shown once for the SysOp to relay.
- **SSH two-factor enforcement:** the SSH `PublicKeyCallback` rejects any
  connection whose offered key is not an ACTIVE key of an `approved` user with
  that handle. After the transport authenticates the key, the app prompts for
  the password (argon2id) — both required, every connection.
- **First-login onboarding:** an approved user with no password set is prompted
  for their one-time token, then sets a password (entered twice).
- **Auth defence (SEC-4):** rate-limit / backoff on password attempts, generic
  "login failed" message, constant-time-ish lookup.
- **Daily time budget:** enforce `daily_minutes` per non-SysOp member, decremented
  from session duration; kick when exhausted.
- **Self-service key management:** an approved member can list / add / revoke
  their own SSH keys from a profile menu.

## Out of scope

- Message boards / mail / files / doors (S004–S007).
- The SysOp Control Panel UI (S008) — S003 provides the approval *operation*
  (CLI/seed path); the rich panel comes later.

## Inputs

- `planning/DECISIONS.md` (auth, 2FA, token delivery), `docs/DATA_MODEL.md`,
  `docs/PERMISSIONS.md`, the data layer shipped on `main`.

## Outputs

- A pending applicant can be approved and can then log in over SSH with key +
  password; pending/unapproved/unknown-key connections are refused.
- Tests + a real SyncTERM/ssh smoke covering onboarding and a repeat login.
