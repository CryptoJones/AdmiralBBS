# journal/

Append-only daily / weekly log. **Complements** `STATE.md` — does not replace it.

- `STATE.md` is the *current* moment — edited in place.
- `journal/YYYY-MM-DD.md` is *what happened* on that date — never edited later.

When you find yourself wondering "what changed between sprints 003 and 005?", this folder is the answer. `git log` is too noisy; STATE.md only holds the latest snapshot. The journal is the middle ground.

## Create today's entry

```bash
socrates journal
```

That creates `journal/YYYY-MM-DD.md` (with a short template) and opens it in `$EDITOR`. Save & quit to commit the entry.

## What to write

One short entry per working day, freeform. Things worth recording:

- What you decided that did not yet make it to `DECISIONS.md`.
- What surprised you (those become tomorrow's risks).
- What the client said in a call.
- What you tried that did not work — and why.

Future builders (human or agent) will read these in order. Optimize for skim-ability, not completeness.
