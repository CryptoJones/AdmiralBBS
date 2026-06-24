# Console Cowboy 2026

A multiplayer cyberpunk MUD in the MajorMUD/Worldgroup tradition, served as a
**resident door** for AdmiralBBS. One persistent server holds one shared world;
the BBS bridges every caller into it, so all players inhabit the same Night City
and the Net beyond it — they see each other, chat, and fight in real time.

## Architecture

- **Engine** (`src/game/cowboy`) — pure, single-threaded, network-free world
  logic (rooms, mobs, combat, progression, shops, quests, persistence). Fully
  unit-tested; combat uses an injectable RNG so it's deterministic under test.
- **Server** (`src/cmd/cowboy`) — a TCP server that owns I/O only. Every state
  change runs on one goroutine (player commands + a world tick are serialized),
  so the engine never needs locks.
- **Bridge** — the BBS `resident` door relays a caller's session bytes to the
  server. N callers bridged into one server = N players in one world.

## Running it

```sh
# 1) start the game server (its own SQLite character DB)
go run ./src/cmd/cowboy -addr 127.0.0.1:4000 -db cowboy.db -tick 2s

# 2) tell the BBS to register it as a resident door pointing at that address
go run ./src/cmd/admiralbbs -cowboy 127.0.0.1:4000   # (plus your usual flags)
```

Members then pick **Console Cowboy 2026** from the Door Games menu and are
bridged straight in. A SysOp can also register/point it from the control panel
(Content → register door → resident).

`-cowboy <addr>` does **not** make the game single-player — it is exactly what
wires up multiplayer: it points the shared resident door at the one running
server. (Subprocess doors spawn one process per player; resident doors share
one server. Console Cowboy is resident.)

## Gameplay

- **Character creation** — new runners pick a class (Netrunner / Solo / Fixer /
  Techie; CP2020/GURPS-flavored base stats) and spend skill points across
  **Body** (melee damage, HP), **Reflexes** (to-hit, dodge), and **Intelligence**
  (netrun breaching).
- **Core loop** — explore Night City and the Net, fight hostiles, earn XP and
  **€$ (eddies)**, level up (cap **level 50**), buy gear, and run fixer bounties.
- **Combat** — MajorMUD-style rounds resolved on the tick: a to-hit roll vs the
  target's Armor Class, then damage = attack − soak (min 1). Mobs respawn on a
  cooldown. Death is never permanent (it's a BBS door) — you respawn in the
  alley at half HP with a small €$/XP penalty.
- **Netrunning** — in the Net, your **ATTACK** is a breach driven by
  **Intelligence**, not a meatspace strike driven by Body.
- **Quests** — fixers (vendor rooms) post repeatable bounties: `quests` to see
  the board, `accept <#>`, kill the target count anywhere, then `claim` back at
  a fixer for XP and eddies.

### Commands

`N S E W U D` movement · `look` · `attack <foe>` · `flee` · `say <msg>` · `who` ·
`score` · `list`/`buy <item>`/`use <item>` · `inventory` · `quests`/`accept <#>`/`claim` ·
`help` · `quit`

## Verified

- Unit tests (`tests/cowboy*_test.go`): connect/look, movement, deterministic
  combat + rewards, multiplayer visibility + chat + who, shop buy/use,
  net-breach routing, persistence, character creation (incl. defaults), quest
  accept→kill→claim, min-level gating, and the level cap.
- Load-tested: **50 unique players** concurrently created characters and shared
  one world (`who` listed all 50; 50 characters persisted across all 4 classes).

---
*Proudly Made in Nebraska. Go Big Red! 🌽 <https://xkcd.com/2347/>*
