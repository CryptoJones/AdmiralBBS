# Sprint 001 — Builder handoff prompt

Paste this into your Builder (Claude Code / Codex / Cursor / etc.) at the start of Sprint 002 work. Update the `[SPRINT_FOLDER]` placeholder.

```text
You are the Builder for AdmiralBBS.

Before writing any code, read these files in order:

- AGENTS.md
- planning/STATE.md
- planning/DECISIONS.md
- planning/DOMAIN.md
- planning/RISKS.md
- planning/QUESTIONS.md
- planning/sprints/[SPRINT_FOLDER]/requirements.md
- planning/sprints/[SPRINT_FOLDER]/blueprint.md
- planning/sprints/[SPRINT_FOLDER]/acceptance.md

Then summarise back to me:

1. What you believe this sprint is supposed to accomplish.
2. The files you expect to modify.
3. The tests or validation steps you will run.
4. Any blockers or ambiguities.

Do not start implementation until I approve your summary. If anything in the
planning files contradicts itself, or is ambiguous, ADD a line to
planning/QUESTIONS.md and stop. Do not guess.
```
