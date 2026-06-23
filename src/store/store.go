// Package store is the data layer: an embedded SQLite database (pure-Go
// modernc driver) behind repository types. Subsystems depend on the repos, not
// on raw SQL. See docs/DATA_MODEL.md and planning/DECISIONS.md.
package store

import (
	"database/sql"
	"embed"
	"encoding/base64"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"admiralbbs/src/crypto"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store owns the database handle and hands out repositories.
type Store struct {
	db    *sql.DB
	vault *crypto.Vault
}

// Open opens (creating if needed) the SQLite database at path with the agreed
// pragmas — WAL journalling, a busy timeout, foreign keys, and NORMAL sync —
// then applies any pending migrations. The vault encrypts sensitive fields at
// rest and is required (encryption is mandatory).
func Open(path string, vault *crypto.Vault) (*Store, error) {
	if vault == nil {
		return nil, fmt.Errorf("store: vault is required (encryption is mandatory)")
	}
	dsn := fmt.Sprintf("file:%s?", url.PathEscape(path)) + strings.Join([]string{
		"_pragma=journal_mode(WAL)",
		"_pragma=busy_timeout(5000)",
		"_pragma=foreign_keys(1)",
		"_pragma=synchronous(1)", // NORMAL
	}, "&")

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &Store{db: db, vault: vault}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// seal encrypts a string for at-rest storage, returning base64 ciphertext.
func (s *Store) seal(plain string) (string, error) {
	ct, err := s.vault.Seal([]byte(plain))
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(ct), nil
}

// open decrypts base64 ciphertext produced by seal.
func (s *Store) open(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	ct, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	plain, err := s.vault.Open(ct)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// DB exposes the handle for advanced callers and tests.
func (s *Store) DB() *sql.DB { return s.db }

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// Users returns the user repository.
func (s *Store) Users() *Users { return &Users{st: s} }

// Memberships returns the membership repository.
func (s *Store) Memberships() *Memberships { return &Memberships{st: s} }

// Keys returns the SSH-key repository.
func (s *Store) Keys() *Keys { return &Keys{st: s} }

// Tokens returns the one-time approval-token repository.
func (s *Store) Tokens() *Tokens { return &Tokens{st: s} }

// SessionLog returns the audit mirror sink (implements audit.Sink).
func (s *Store) SessionLog() *SessionLog { return &SessionLog{st: s} }

// migrate applies embedded migrations whose number exceeds the database's
// current PRAGMA user_version, each in its own transaction, in order.
func (s *Store) migrate() error {
	current, err := s.userVersion()
	if err != nil {
		return err
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		ver, err := migrationVersion(name)
		if err != nil {
			return err
		}
		if ver <= current {
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", name, err)
		}
		// user_version cannot be set via a bound parameter.
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", ver)); err != nil {
			tx.Rollback()
			return fmt.Errorf("set user_version %d: %w", ver, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		current = ver
	}
	return nil
}

func (s *Store) userVersion() (int, error) {
	var v int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return 0, err
	}
	return v, nil
}

// migrationVersion parses the leading integer of a migration filename
// ("001_init.sql" -> 1).
func migrationVersion(name string) (int, error) {
	i := strings.IndexByte(name, '_')
	if i <= 0 {
		return 0, fmt.Errorf("migration %q must be <number>_<name>.sql", name)
	}
	n, err := strconv.Atoi(name[:i])
	if err != nil {
		return 0, fmt.Errorf("migration %q: bad version: %w", name, err)
	}
	return n, nil
}
