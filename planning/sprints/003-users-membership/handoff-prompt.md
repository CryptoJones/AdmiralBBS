# Sprint 003 — Builder handoff prompt

```text
You are the Builder for AdmiralBBS, Sprint 003 (Users & Membership / 2FA auth).

Read in order: AGENTS.md, planning/STATE.md, planning/DECISIONS.md,
planning/DOMAIN.md, planning/RISKS.md, planning/QUESTIONS.md,
docs/DATA_MODEL.md, docs/PERMISSIONS.md, and
planning/sprints/003-users-membership/{requirements,blueprint,acceptance}.md

Summarise back: the goal, the files you'll touch, the tests/validation, and any
ambiguity. Do not start until I approve. If anything contradicts, add a line to
planning/QUESTIONS.md and stop.

Key invariants: SSH = two factors (registered key AND password) on EVERY
connection; Telnet stays apply-only; the one-time token is single-use,
time-limited, stored hashed, delivered out-of-band; encryption-at-rest and the
audit hash-chain from the data layer must keep working.
```
