<p align="center"><em>Proudly Made in Nebraska. Go Big Red! 🌽 <a href="https://xkcd.com/2347/">https://xkcd.com/2347/</a></em></p>

# AdmiralBBS

Clean-room implementation of 90's era ANSI BBSes

Client: **CryptoJones**

## 🔐 Security was a founding decision, not a bolt-on

AdmiralBBS was designed encrypted from sprint zero. These are architectural
commitments recorded in [`planning/DECISIONS.md`](planning/DECISIONS.md) before
the first line of feature code — not retrofitted later:

- **Encryption in transit.** SSH for everything. Telnet is permitted *only* for
  the membership-application screen; once you're a member, all access is over
  SSH. No password or secret ever crosses a plaintext channel.
- **Encryption at rest (two layers).** Sensitive payloads — message and mail
  bodies, file-library contents, PII, and the audit log — are sealed with
  **XChaCha20-Poly1305**; the key is derived (Argon2id) from a startup secret,
  held only in memory (`mlock`'d, zeroed on exit), and **never written to the
  data volume**. Underneath, the whole data directory runs on an **encrypted
  volume** so even structural metadata is ciphertext on a stolen disk.
- **Memory-safe core.** Written in Go — the buffer-overflow class is gone by
  construction. Door games run as **sandboxed subprocesses** (separate uid,
  jail, scrubbed environment) so they can never reach the host or the key.
- **Honest threat model.** This protects fully against offline access — a
  stolen disk, copied volume, image layer, backup, or stopped container is
  unreadable without the key. It raises but does not eliminate the bar against
  an attacker with *live root* on the running host (who could scrape the key
  from process memory); fully closing that needs hardware (TPM/HSM/enclave),
  which is out of scope for this stack. See
  [`planning/RISKS.md`](planning/RISKS.md).

## Why this exists

BBSes were apart of a lot of hackers childhoods and this is a fun project to pay homage to that.

## Status

See [`planning/STATE.md`](planning/STATE.md) for the current sprint and next action.

## How to navigate

- **`AGENTS.md`** — start here. The tool-agnostic project router.
- **`planning/`** — the operating system (decisions, domain, risks, sprints).
- **`docs/`** — living architecture & validation reference.
- **`src/`** — implementation.

## Methodology

This project follows the [120x Operators Kit](https://120x.ai) Architect/Builder methodology — Architect thinks and writes the plan; Builder reads the plan and writes the code; the handoff is a folder, not a conversation.

## Thanks 🙏

In the spirit of [xkcd 2347](https://xkcd.com/2347/), AdmiralBBS stands on the
shoulders of the cryptographers whose work makes "secure from sprint zero"
possible at all. We didn't invent any of this — we just get to use it:

- **Mihir Bellare, Ran Canetti & Hugo Krawczyk** — HMAC (1996), which makes our
  audit trail tamper-evident.
- **Ralph Merkle** — hash chains / hash trees, the idea of committing each
  record to the one before it.
- **Daniel J. Bernstein** — ChaCha20 and Poly1305, the backbone of the
  XChaCha20-Poly1305 encryption protecting data at rest.
- **Alex Biryukov, Daniel Dinu & Dmitry Khovratovich** — Argon2 (Password
  Hashing Competition winner), used both to hash member passwords and to derive
  the master key.
- The **Go team** and **`golang.org/x/crypto`** maintainers for trustworthy,
  audited implementations of all of the above.

To everyone maintaining the unglamorous crypto primitives the whole internet
quietly depends on: thank you.
