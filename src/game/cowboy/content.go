package cowboy

// startRoom is where new and respawning cowboys appear — a PRIVATE capsule pod,
// so a fresh jack-in or a respawn can never be spawn-camped. You step OUT into
// the street (Neon Alley) under your own power.
const startRoom = "capsule"

// buildRooms returns the Chrome Circuit Cowboys world map — a slice of the city
// and the Net beyond the jack-in port.
func buildRooms() map[string]*Room {
	rooms := []*Room{
		{ID: "capsule", Name: "Re-Clone Bay :: Your Booth", Private: true, Safe: true,
			Desc: "A private booth in the clone clinic. You come to in a fresh clone, your\r\nmind restored from its realtime backup — calm, whole, and a few scrip lighter\r\nfor the new body. The clinic doors slide OUT into the street.",
			Exits: map[string]string{"out": "neon_alley"}},
		{ID: "neon_alley", Name: "Neon Alley", Safe: true,
			Desc: "Rain hisses on hot neon. Holo-ads for synth-ramen and combat clinics\r\nflicker across puddles. The Strip roars to the east; a battered door to\r\nthe south leads into the Chrome Rose; the re-clone clinic is just IN off\r\nthe street. Security drones hum overhead — draw on another runner here and they\r\nflatline you on the spot. (no-violence zone)",
			Exits: map[string]string{"east": "the_sprawl", "south": "chrome_bar", "in": "capsule"}},
		{ID: "chrome_bar", Name: "The Chrome Rose", Vendor: true,
			Desc: "A runner dive. Chrome-plated regulars jack into the bar's local node\r\nwhile an augmented bartender slings stims and gear. A vendor terminal glows\r\nhere (type LIST).",
			Exits: map[string]string{"north": "neon_alley"}},
		{ID: "the_sprawl", Name: "The Strip",
			Desc: "Endless arcologies stacked into the smog. Crowds churn between street\r\nstalls. A black alley opens north; corporate spires gleam east; the Night\r\nMarket is south.",
			Exits: map[string]string{"west": "neon_alley", "north": "back_alley", "east": "corpo_plaza", "south": "market"}},
		{ID: "back_alley", Name: "Back Alley",
			Desc: "A dead-end choked with dumpsters and busted drones. Gangers tag the walls\r\nin UV paint and don't like tourists.",
			Exits: map[string]string{"south": "the_sprawl"}},
		{ID: "market", Name: "Night Market", Vendor: true, Medic: true,
			Desc: "Stalls of grey-market cyberware and noodle carts under string lights. A\r\nbroker runs a vendor stall (LIST) and a back-room Emergency Medic wires in chrome\r\n(INSTALL salvaged cyberware).",
			Exits: map[string]string{"north": "the_sprawl"}},
		{ID: "corpo_plaza", Name: "Corporate Plaza",
			Desc: "Glass and gun-metal. Security drones sweep the concourse and corpo-sec in\r\nmirror visors watch everything. A guarded data port hums to the east.",
			Exits: map[string]string{"west": "the_sprawl", "east": "data_port"}},
		{ID: "data_port", Name: "Data Port",
			Desc: "A jack-in cradle wired to the city grid. Jacking in (UP) drops your\r\nconsciousness into the Net.",
			Exits: map[string]string{"west": "corpo_plaza", "up": "the_net"}},
		{ID: "the_net", Name: "The Net :: Grid Node",
			Desc: "Wireframe canyons of glowing data. White ICE patrols the lattice. A\r\nguarded gateway descends (DOWN) toward the Sentinel Lattice.",
			Exits: map[string]string{"down": "ice_wall", "up": "data_port"}},
		{ID: "ice_wall", Name: "The Net :: Sentinel Lattice",
			Desc: "A churning wall of layered ICE seals the way down. A single Gauntlet\r\nconstruct reconfigures itself endlessly — beat one shell and a harder one\r\nrises. Past it (DOWN) lies the Black ICE Fortress. This deep, runners can\r\njack each other: PvP is live here.",
			Exits: map[string]string{"up": "the_net", "down": "deep_net"}},
		{ID: "deep_net", Name: "Deep Net :: Black ICE Fortress",
			Desc: "The architecture turns predatory. Black ICE coils in the dark and the\r\nRogue AI watches from the core. Runners fight each other here as readily\r\nas the ICE — PvP is live.",
			Exits: map[string]string{"up": "ice_wall"}},
	}
	m := make(map[string]*Room, len(rooms))
	for _, r := range rooms {
		m[r.ID] = r
	}
	return m
}

// mobTemplates defines the hostiles and where they live. The Home field is set
// from the map key for respawn placement.
func buildMobTemplates() map[string]*MobTemplate {
	defs := []*MobTemplate{
		{ID: "ganger", Name: "a street ganger", HP: 18, Damage: 4, AC: 2, XP: 25, Eddies: 10, Aggressive: true, Home: "back_alley"},
		{ID: "drone", Name: "a security drone", HP: 30, Damage: 7, AC: 5, XP: 50, Eddies: 25, Aggressive: false, Home: "corpo_plaza"},
		{ID: "corposec", Name: "a corpo-sec officer", HP: 45, Damage: 10, AC: 6, XP: 80, Eddies: 40, Aggressive: true, Home: "corpo_plaza"},
		{ID: "white_ice", Name: "a White ICE sentinel", HP: 35, Damage: 9, AC: 5, XP: 70, Eddies: 30, Aggressive: true, Home: "the_net"},
		{ID: "black_ice", Name: "a Black ICE daemon", HP: 80, Damage: 16, AC: 8, XP: 200, Eddies: 120, Aggressive: true, Home: "deep_net"},
		{ID: "rogue_ai", Name: "the Rogue AI", HP: 150, Damage: 22, AC: 10, XP: 500, Eddies: 400, Aggressive: true, Home: "deep_net"},
		// Multi-stage ICE: only the white shell spawns (Home set); on "death" each
		// stage morphs into the next, harder one. Only the final lock pays out.
		{ID: "gauntlet1", Name: "the Gauntlet ICE [white shell]", HP: 40, Damage: 10, AC: 5, Aggressive: true, Home: "ice_wall", Next: "gauntlet2"},
		{ID: "gauntlet2", Name: "the Gauntlet ICE [black core]", HP: 70, Damage: 16, AC: 8, Aggressive: true, Next: "gauntlet3"},
		{ID: "gauntlet3", Name: "the Gauntlet ICE [lethal lock]", HP: 110, Damage: 24, AC: 11, XP: 700, Eddies: 600, Aggressive: true},
	}
	m := make(map[string]*MobTemplate, len(defs))
	for _, t := range defs {
		m[t.ID] = t
	}
	return m
}

// ware is a purchasable item.
type ware struct {
	name  string
	price int
	heal  int // stimpak: HP restored on use
	ram   int // ram-chip: RAM restored on use
	bonus int // weapon: attack bonus granted on purchase (permanent)
	deck  int // cyberdeck: MaxRAM bonus granted on purchase (permanent)
	desc  string
}

// shopWares are sold at any Vendor room.
var shopWares = []ware{
	{name: "stimpak", price: 20, heal: 25, desc: "single-use trauma stim, restores 25 HP"},
	{name: "ram-chip", price: 30, ram: 8, desc: "single-use RAM chip, restores 8 RAM for netruns"},
	{name: "ice-breaker", price: 150, bonus: 5, desc: "intrusion blade, +5 attack (permanent)"},
	{name: "mono-katana", price: 400, bonus: 12, desc: "monomolecular katana, +12 attack (permanent)"},
	{name: "cyberdeck", price: 250, deck: 8, desc: "upgraded deck, +8 max RAM (permanent)"},
}

func findWare(name string) (ware, bool) {
	for _, w := range shopWares {
		if w.name == name {
			return w, true
		}
	}
	return ware{}, false
}
