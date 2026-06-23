# RISKS — known traps

A short, living list. When a risk materialises, move it to `DECISIONS.md` with the mitigation chosen.

## Risks

- mythos level AI systems that can login to the BBS and and escape into the parent OS

## Input fragility

With terminal emulation there were a variety of attack surfaces that hackers exploited. We need to harden against all known and unknown attacks of this nature.

## Always-on risks for any 120x project

- AI output is not source of truth. Numbers must trace back to data, documents, or human confirmation.
- Single-file overload — context must be split across the planning files, not crammed into one.
- Tool churn — the methodology must survive any specific agent going away.
