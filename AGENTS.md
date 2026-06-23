# AGENTS.md — AdmiralBBS

> Tool-agnostic router. **Read this first** before making any changes.

## What this project is

**AdmiralBBS** — Clean-room implementation of 90's era ANSI BBSes

Client: **CryptoJones**

Tech stack: **TBD**

## Repo layout

```text
AdmiralBBS/
├── AGENTS.md            ← you are here
├── CLAUDE.md            ← thin adapter for Claude Code
├── CODEX.md             ← thin adapter for Codex
├── README.md            ← human-facing overview
├── docs/                ← living architecture / data model / validation
├── planning/            ← the operating system for this project
│   ├── STATE.md         ← current sprint + next action
│   ├── DECISIONS.md     ← durable choices ("the house rules")
│   ├── DOMAIN.md        ← client terminology + workflow
│   ├── RISKS.md         ← known traps
│   ├── QUESTIONS.md     ← unresolved items
│   ├── FILE_INVENTORY.md
│   └── sprints/         ← one folder per sprint
├── src/                 ← application code
├── tests/               ← tests
├── scripts/             ← one-off / ops scripts
├── samples/             ← sample inputs (real or anonymised)
└── references/          ← supporting docs from the client
```

## How to start work

1. Read **`planning/STATE.md`** — what sprint we are in and what is next.
2. Read **`planning/DECISIONS.md`** and **`planning/DOMAIN.md`** — context that must not be re-derived.
3. Read the active sprint folder: **`planning/sprints/<active>/`** — `requirements.md`, `blueprint.md`, `acceptance.md`.
4. Confirm scope back to the operator **before** writing code.

## Rules

- Do not redefine scope. If a sprint requirement is ambiguous, add it to `planning/QUESTIONS.md` and stop.
- Do not invent business rules. They belong in `planning/DOMAIN.md` and `planning/DECISIONS.md`.
- Update `planning/STATE.md` at the end of every working session.
- New durable choices go to `planning/DECISIONS.md`.
- Validation is against real business expectations, not just passing tests — see `docs/VALIDATION.md`.

## Handoff principle

> The handoff is a folder, not a conversation.

The chat history is not the source of truth. The folder is.
