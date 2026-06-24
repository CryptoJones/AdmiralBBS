package cowboy

import (
	"database/sql"
	"encoding/json"

	_ "modernc.org/sqlite"
)

// SQLiteStore persists characters in a pure-Go SQLite file (separate from the
// BBS database — the game owns its own state).
type SQLiteStore struct{ db *sql.DB }

// OpenSQLite opens (or creates) the character database at path.
func OpenSQLite(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS cowboy_player (
		name          TEXT PRIMARY KEY COLLATE NOCASE,
		class         TEXT NOT NULL DEFAULT '',
		level         INTEGER NOT NULL,
		xp            INTEGER NOT NULL,
		eddies        INTEGER NOT NULL,
		hp            INTEGER NOT NULL,
		maxhp         INTEGER NOT NULL,
		body          INTEGER NOT NULL,
		reflexes      INTEGER NOT NULL,
		intelligence  INTEGER NOT NULL,
		weapon_bonus  INTEGER NOT NULL,
		weapon_name   TEXT NOT NULL,
		room          TEXT NOT NULL,
		inv_json      TEXT NOT NULL,
		quests_json   TEXT NOT NULL DEFAULT '{}'
	)`); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

// Close releases the database.
func (s *SQLiteStore) Close() error { return s.db.Close() }

// Load fetches a saved character by name.
func (s *SQLiteStore) Load(name string) (*SavedPlayer, bool, error) {
	var sp SavedPlayer
	var invJSON, questsJSON string
	err := s.db.QueryRow(`SELECT name, class, level, xp, eddies, hp, maxhp, body, reflexes,
		intelligence, weapon_bonus, weapon_name, room, inv_json, quests_json
		FROM cowboy_player WHERE name = ? COLLATE NOCASE`, name).
		Scan(&sp.Name, &sp.Class, &sp.Level, &sp.XP, &sp.Eddies, &sp.HP, &sp.MaxHP, &sp.Body,
			&sp.Reflexes, &sp.Intelligence, &sp.WeaponBonus, &sp.WeaponName, &sp.Room, &invJSON, &questsJSON)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	sp.Inv = map[string]int{}
	_ = json.Unmarshal([]byte(invJSON), &sp.Inv)
	sp.Quests = map[string]int{}
	_ = json.Unmarshal([]byte(questsJSON), &sp.Quests)
	return &sp, true, nil
}

// Save upserts a character.
func (s *SQLiteStore) Save(sp *SavedPlayer) error {
	inv, _ := json.Marshal(sp.Inv)
	qjson, _ := json.Marshal(sp.Quests)
	_, err := s.db.Exec(`INSERT INTO cowboy_player
		(name, class, level, xp, eddies, hp, maxhp, body, reflexes, intelligence, weapon_bonus, weapon_name, room, inv_json, quests_json)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(name) DO UPDATE SET
		  class=excluded.class, level=excluded.level, xp=excluded.xp, eddies=excluded.eddies, hp=excluded.hp,
		  maxhp=excluded.maxhp, body=excluded.body, reflexes=excluded.reflexes,
		  intelligence=excluded.intelligence, weapon_bonus=excluded.weapon_bonus,
		  weapon_name=excluded.weapon_name, room=excluded.room, inv_json=excluded.inv_json,
		  quests_json=excluded.quests_json`,
		sp.Name, sp.Class, sp.Level, sp.XP, sp.Eddies, sp.HP, sp.MaxHP, sp.Body, sp.Reflexes,
		sp.Intelligence, sp.WeaponBonus, sp.WeaponName, sp.Room, string(inv), string(qjson))
	return err
}
