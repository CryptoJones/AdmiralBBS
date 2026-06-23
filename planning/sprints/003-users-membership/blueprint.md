# Sprint 003 — Users & Membership (2FA auth) | blueprint

## Files to create / modify

| Path | Purpose |
|---|---|
| `src/store/migrations/002_approval_tokens.sql` | `approval_token` table (token_hash, expires_at, used_at) |
| `src/store/tokens.go` | `Tokens` repo: `Issue(userID)` → plaintext (stores hash); `Redeem(userID, plaintext)` |
| `src/store/users.go` | `Approve(userID, level)`; `SetPassword` already exists |
| `src/transport/ssh.go` | flip to `PublicKeyCallback`; add an `Authenticator` hook |
| `src/transport/conn.go`, `src/session/session.go` | expose `Username()` on the session |
| `src/menu/login.go` | SSH login/onboarding flow (token → set password → / verify password) |
| `src/menu/profile.go` | self-service SSH-key management (list/add/revoke) |
| `src/cmd/admiralbbs/main.go` | wire the SSH authenticator + run login before the menu; enforce daily budget |
| `tests/*` | tokens, login/onboarding, daily-budget, authz |

## Auth design (the two factors)

```text
SSH connect (handle = ssh user)
  │  factor 1 — transport: PublicKeyCallback
  │    look up user by handle; must be `approved`; offered key must match an
  │    ACTIVE user_key (Keys.Authorizes). Else: reject the handshake.
  ▼
App login flow (encrypted channel)
  │  if password_hash == "" : ONBOARDING
  │     prompt one-time token → Tokens.Redeem → set password (twice) → store hash
  │  else : prompt password → VerifyPassword  (factor 2 — something you know)
  │     SEC-4: backoff after failures, generic "login failed"
  ▼
  daily-budget check → main menu
```

## Step-by-step

1. Migration 002 + `Tokens` repo (hash = sha256 of a 32-byte random token;
   expiry default 72h; single-use via `used_at`). Tests.
2. `Users.Approve` (status=approved, access_level). Approval issues a token.
3. `session.Username()` accessor; `menu.RunLogin`.
4. Flip SSH to `PublicKeyCallback` via an `Authenticator` hook; main wires it to
   the store. Pending/unknown/with-wrong-key handshakes are rejected.
5. `main` SSH handler: `RunLogin` → on success enforce daily budget → menu.
6. `menu.RunProfile` self-service key add/revoke.
7. Tests + SyncTERM/ssh smoke (onboard once, reconnect with password).

## Notes

- Token plaintext is returned to the SysOp once (relayed out-of-band); only the
  hash is stored. Never log the plaintext.
- Telnet path is unchanged (apply-only).
- Daily budget: SysOp (level ≥ 100) is unlimited; others use `daily_minutes`
  (default 60, configurable) — see DECISIONS.
