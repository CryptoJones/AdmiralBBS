# Sprint 003 — Users & Membership (2FA auth) | acceptance

## Functional

- [ ] A pending applicant (from Telnet apply) can be approved; approval yields a
      one-time token shown once.
- [ ] An approved member connects over SSH **only** if their offered key matches
      an active registered key; pending / unknown / wrong-key handshakes are
      rejected at the transport.
- [ ] First SSH login: prompts for the one-time token, then sets a password.
      Token is single-use and time-limited (re-use / expiry rejected).
- [ ] Subsequent SSH logins require the password (2nd factor); wrong password is
      rejected with a generic message and backoff.
- [ ] Daily time budget is enforced for non-SysOp members.
- [ ] A member can list / add / revoke their own SSH keys; a revoked key no
      longer authenticates.

## Security (negative tests)

- [ ] A valid key with a wrong password fails (and vice-versa) — both factors
      required.
- [ ] Expired or already-used token is rejected.
- [ ] Login errors do not reveal whether a handle exists (generic message).

## Process

- [ ] `go build` / `vet` / `test` green; real ssh smoke (onboard + reconnect).
- [ ] `planning/STATE.md` updated; new decisions recorded.
