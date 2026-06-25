package store

import (
	"database/sql"
	"errors"
	"time"
)

// InstalledDoor is a resident door the BBS installed from a forge release URL
// (binary downloaded, run under supervision, bridged). Persisted so it can be
// relaunched and re-registered on restart.
type InstalledDoor struct {
	ID             int64
	Name           string
	SourceURL      string
	Version        string
	BinPath        string
	Address        string
	MinAccessLevel int
}

// InstalledDoors is the repository of release-installed doors.
type InstalledDoors struct{ st *Store }

// InstalledDoors returns the release-installed-door repository.
func (s *Store) InstalledDoors() *InstalledDoors { return &InstalledDoors{st: s} }

// Upsert records (or updates, by name) an installed door.
func (r *InstalledDoors) Upsert(d InstalledDoor, at time.Time) error {
	_, err := r.st.db.Exec(`
		INSERT INTO installed_door (name, source_url, version, bin_path, address, min_access_level, installed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			source_url       = excluded.source_url,
			version          = excluded.version,
			bin_path         = excluded.bin_path,
			address          = excluded.address,
			min_access_level = excluded.min_access_level,
			installed_at     = excluded.installed_at`,
		d.Name, d.SourceURL, d.Version, d.BinPath, d.Address, d.MinAccessLevel, fmtTime(at))
	return err
}

// List returns all installed doors, ordered by name.
func (r *InstalledDoors) List() ([]InstalledDoor, error) {
	rows, err := r.st.db.Query(
		`SELECT id, name, source_url, version, bin_path, address, min_access_level
		 FROM installed_door ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InstalledDoor
	for rows.Next() {
		var d InstalledDoor
		if err := rows.Scan(&d.ID, &d.Name, &d.SourceURL, &d.Version, &d.BinPath, &d.Address, &d.MinAccessLevel); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ByName returns one installed door, or (nil, nil) if absent.
func (r *InstalledDoors) ByName(name string) (*InstalledDoor, error) {
	var d InstalledDoor
	err := r.st.db.QueryRow(
		`SELECT id, name, source_url, version, bin_path, address, min_access_level
		 FROM installed_door WHERE name = ?`, name).
		Scan(&d.ID, &d.Name, &d.SourceURL, &d.Version, &d.BinPath, &d.Address, &d.MinAccessLevel)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

// Remove deletes an installed door record by name (the binary/file is removed by
// the caller).
func (r *InstalledDoors) Remove(name string) error {
	_, err := r.st.db.Exec(`DELETE FROM installed_door WHERE name = ?`, name)
	return err
}
