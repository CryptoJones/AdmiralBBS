# Sprint 002 — Builder handoff prompt

Paste this into your Builder (Claude Code / Codex / Cursor / etc.) to start
Sprint 002 work.

```text
You are the Builder for AdmiralBBS.

Before writing any code, read these files in order:

- AGENTS.md
- planning/STATE.md
- planning/DECISIONS.md
- planning/DOMAIN.md
- planning/RISKS.md
- planning/QUESTIONS.md
- planning/SprintPlanning.md
- docs/ARCHITECTURE.md
- docs/DATA_MODEL.md
- docs/VALIDATION.md
- planning/sprints/002-core-session-engine/requirements.md
- planning/sprints/002-core-session-engine/blueprint.md
- planning/sprints/002-core-session-engine/acceptance.md

Then summarise back to me:

1. What you believe this sprint is supposed to accomplish.
2. The files you expect to create/modify.
3. The tests and the real-client validation steps you will run.
4. Any blockers or ambiguities.

Do not start implementation until I approve your summary. If anything in the
planning files contradicts itself, or is ambiguous, ADD a line to
planning/QUESTIONS.md and stop. Do not guess — especially do not invent
business rules (time limits, membership rules) that belong to the operator.

This sprint is the spine: Telnet + SSH transports, a transport-agnostic
Session with a HARDENED input reader, terminal detection (ANSI vs B&W, CP437),
and a data-driven menu engine rendering CP437 .ans art. Stack is Go; ship the
hardening (bounded reader + escape validation) WITH its negative tests in this
sprint, not later.
```
