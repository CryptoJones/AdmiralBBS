package store

import (
	"database/sql"
	"errors"
	"os"
)

// Door is a registered door game.
type Door struct {
	ID             int64
	Name           string
	Command        string
	DropfileFormat string
	MinAccessLevel int
}

// Doors is the door-game repository.
type Doors struct{ st *Store }

func (r *Doors) Create(name, command, dropfileFormat string, minLevel int) (*Door, error) {
	if dropfileFormat == "" {
		dropfileFormat = "door32.sys"
	}
	res, err := r.st.db.Exec(
		`INSERT INTO door (name, command, dropfile_format, min_access_level) VALUES (?, ?, ?, ?)`,
		name, command, dropfileFormat, minLevel)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Door{ID: id, Name: name, Command: command, DropfileFormat: dropfileFormat, MinAccessLevel: minLevel}, nil
}

func (r *Doors) Count() (int, error) {
	var n int
	err := r.st.db.QueryRow(`SELECT COUNT(*) FROM door`).Scan(&n)
	return n, err
}

func (r *Doors) scan(row interface{ Scan(...any) error }) (*Door, error) {
	var d Door
	if err := row.Scan(&d.ID, &d.Name, &d.Command, &d.DropfileFormat, &d.MinAccessLevel); err != nil {
		return nil, err
	}
	return &d, nil
}

const doorCols = `id, name, command, dropfile_format, min_access_level`

// Visible lists doors the access level may launch.
func (r *Doors) Visible(accessLevel int) ([]*Door, error) {
	rows, err := r.st.db.Query(`SELECT `+doorCols+` FROM door WHERE min_access_level <= ? ORDER BY name`, accessLevel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Door
	for rows.Next() {
		d, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ByID fetches one door, enforcing access level.
func (r *Doors) ByID(id int64, accessLevel int) (*Door, error) {
	row := r.st.db.QueryRow(`SELECT `+doorCols+` FROM door WHERE id = ?`, id)
	d, err := r.scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if accessLevel < d.MinAccessLevel {
		return nil, ErrNotFound
	}
	return d, nil
}

// Doors returns the door repository.
func (s *Store) Doors() *Doors { return &Doors{st: s} }

// EnsureSeedDoors registers the bundled demo door on first run, if its script is
// present relative to the working directory.
func (s *Store) EnsureSeedDoors() error {
	n, err := s.Doors().Count()
	if err != nil || n > 0 {
		return err
	}
	const demo = "doors/numguess.sh"
	if _, statErr := os.Stat(demo); statErr != nil {
		return nil // no bundled door available; SysOp can register one later
	}
	_, err = s.Doors().Create("Number Guess", demo, "door32.sys", 0)
	return err
}
