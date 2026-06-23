# QUESTIONS — unresolved items

When a question is answered, **move** the answer into `DECISIONS.md`, `DOMAIN.md`, or the relevant sprint file — do not leave answered questions sitting here.

## Open

- All the features and utilities that BBSes have offered in the past — _partially answered: see the BBS landscape table in the Sprint 001 review (Synchronet/Mystic/ENiGMA/WWIV/LBBS) and the roadmap in `SprintPlanning.md`. Keep open as a backlog for "nice to have" features beyond the four core ones in DECISIONS.md._
- the legal aspects of letting others inside my information systems — _operator/legal decision, not an engineering blocker; revisit before any public deployment (Sprint 009 hardening / go-live)._

## Closed (for traceability)

- **How long per day each user should be allowed to login (in minutes)** — resolved 2026-06-23: default 60 min/day, configurable per-user + server flag, SysOp unlimited (see `DECISIONS.md`).
- **How to work the manual approval process for membership applications** — resolved across 2026-06-23: Telnet apply (handle + SSH keys + contact + note) → SysOp approves → one-time token relayed out-of-band (PGP-email recommended) → first SSH login sets password (see `DECISIONS.md`, SEC-2/13, Sprint 003 plan).
