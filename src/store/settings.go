package store

import "database/sql"

// Settings is the SysOp-editable key/value config (branding + MOTD). Unset keys
// return the provided default, so the BBS works out of the box and a SysOp only
// stores the values they actually change.
type Settings struct{ st *Store }

// Settings returns the settings repository.
func (s *Store) Settings() *Settings { return &Settings{st: s} }

// Get returns the value for key, or def if it isn't set.
func (r *Settings) Get(key, def string) string {
	var v string
	err := r.st.db.QueryRow(`SELECT value FROM setting WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows || err != nil {
		return def
	}
	return v
}

// Set upserts a setting value.
func (r *Settings) Set(key, value string) error {
	_, err := r.st.db.Exec(
		`INSERT INTO setting (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// Default branding values (a SysOp overrides them via the control panel).
const (
	DefaultBBSName = "AdmiralBBS"
	DefaultTagline = "A clean-room '90s-era ANSI Bulletin Board System"
)

// BBSName is the configured BBS name (or the default).
func (r *Settings) BBSName() string { return r.Get("bbs_name", DefaultBBSName) }

// Tagline is the configured tagline (or the default).
func (r *Settings) Tagline() string { return r.Get("tagline", DefaultTagline) }

// MOTD is the configured message of the day ("" = none).
func (r *Settings) MOTD() string { return r.Get("motd", "") }
