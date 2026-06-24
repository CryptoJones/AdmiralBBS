// Package cowboy is the engine for "Console Cowboy 2026", a multiplayer
// cyberpunk MUD in the MajorMUD/Worldgroup tradition. It runs as a persistent
// resident door: one shared world, many simultaneous players, bridged in by
// AdmiralBBS. The engine itself is single-threaded and network-free — the
// server (cmd/cowboy) serializes all access on one goroutine and owns I/O, so
// the engine is deterministic and unit-testable.
package cowboy

// Player is a connected console cowboy (netrunner).
type Player struct {
	ID           int
	Name         string
	Class        string
	RoomID       string
	HP, MaxHP    int
	Level, XP    int
	Eddies       int
	Body         int // melee/breach damage
	Reflexes     int // dodge / damage reduction
	Intelligence int // (flavor + future deck mechanics)
	WeaponBonus  int // from a purchased weapon (e.g. ICE-breaker)
	WeaponName   string
	RAM          int // netrun resource: powers breaches in the Net; regenerates out of combat
	DeckBonus    int // bonus MaxRAM from a purchased cyberdeck (permanent)
	Inv          map[string]int // item name -> qty
	Quests       map[string]int // active questID -> kills so far (>= Count means ready to claim)
	fighting     *Mob           // current mob target (nil = not in mob combat)
	pvpTarget    *Player        // current PvP target in the Net (nil = not duelling)
	party        *Party         // co-op crew (nil = solo)
	shieldTicks  int            // remaining ticks of the mirror program's damage shield
	shieldAmt    int            // flat damage reduction while shielded
	dirty        bool           // output was sent since the last prompt (gates tick re-prompts)
	out          func(string)   // content output sink (set by the server; nil-safe via send)
	prompter     func(string)   // optional dedicated prompt sink (managed-prompt I/O); falls back to out
}

// attack is the player's deterministic damage per round.
func (p *Player) attack() int { return 3 + p.Body/2 + p.Level + p.WeaponBonus }

// defense reduces incoming damage (floored to 1 by the caller).
func (p *Player) defense() int { return p.Reflexes / 4 }

func (p *Player) send(s string) {
	if p.out != nil {
		p.out(s)
	}
	p.dirty = true // output emitted; a fresh prompt is owed (cleared by sendPrompt)
}

// MobTemplate is the static definition of a hostile program/NPC.
type MobTemplate struct {
	ID         string
	Name       string
	HP         int
	Damage     int // attack power
	AC         int // armor class (to-hit difficulty + light damage soak)
	XP         int
	Eddies     int
	Aggressive bool   // attacks players on sight
	Home       string
	Next       string // multi-stage ICE: on "death" it morphs into this template instead of dying
}

// Mob is a live instance of a MobTemplate in the world.
type Mob struct {
	tmpl      *MobTemplate // current template (may be a later stage after a morph)
	origin    *MobTemplate // the spawn template — restored on respawn so multi-stage ICE resets
	HP        int
	RoomID    string
	target    *Player
	respawnIn int // ticks until this dead mob respawns (0 = alive)
	dead      bool
}

// Room is one location in the city/net.
type Room struct {
	ID     string
	Name   string
	Desc   string
	Exits  map[string]string // direction -> room id
	Vendor bool              // a shop operates here
}

// SavedPlayer is the persisted slice of a Player (progress survives logout).
type SavedPlayer struct {
	Name                         string
	Class                        string
	Level, XP, Eddies, HP, MaxHP int
	Body, Reflexes, Intelligence int
	WeaponBonus                  int
	WeaponName                   string
	RAM                          int
	DeckBonus                    int
	Room                         string
	Inv                          map[string]int
	Quests                       map[string]int
}

// Persistence stores character progress between sessions. The server backs it
// with SQLite; tests use an in-memory implementation.
type Persistence interface {
	Load(name string) (*SavedPlayer, bool, error)
	Save(sp *SavedPlayer) error
	// Top returns up to n characters ranked by level then XP (for the leaderboard).
	Top(n int) ([]SavedPlayer, error)
}

// MemStore is an in-memory Persistence for tests and ephemeral runs.
type MemStore struct{ m map[string]*SavedPlayer }

// NewMemStore builds an empty in-memory store.
func NewMemStore() *MemStore { return &MemStore{m: map[string]*SavedPlayer{}} }

// Load returns a saved character by name.
func (s *MemStore) Load(name string) (*SavedPlayer, bool, error) {
	sp, ok := s.m[name]
	return sp, ok, nil
}

// Save upserts a character.
func (s *MemStore) Save(sp *SavedPlayer) error {
	cp := *sp
	s.m[sp.Name] = &cp
	return nil
}

// Top returns up to n characters ranked by level then XP.
func (s *MemStore) Top(n int) ([]SavedPlayer, error) {
	out := make([]SavedPlayer, 0, len(s.m))
	for _, sp := range s.m {
		out = append(out, *sp)
	}
	sortSavedByRank(out)
	if len(out) > n {
		out = out[:n]
	}
	return out, nil
}
