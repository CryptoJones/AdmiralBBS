package store

import (
	"database/sql"
	"fmt"
	"net"
	"strings"
	"time"
)

// Ban is one IP/CIDR ban. Patterns are exact IPs ("203.0.113.7") or CIDR
// blocks ("203.0.113.0/24"); both transports reject matching sources at accept
// time, before authentication. Lifting is soft (lifted_at set) so the record of
// who banned what survives.
type Ban struct {
	ID       int64
	Pattern  string
	Reason   string
	BannedBy sql.NullInt64
	BannedAt time.Time
	LiftedAt *time.Time
}

// Bans is the IP-banlist repository.
type Bans struct{ st *Store }

// Bans returns the banlist repository.
func (s *Store) Bans() *Bans { return &Bans{st: s} }

// NormalizeBanPattern validates and canonicalises a ban pattern. It accepts a
// bare IP (v4 or v6) or a CIDR block, returning the canonical string to store.
func NormalizeBanPattern(pattern string) (string, error) {
	p := strings.TrimSpace(pattern)
	if p == "" {
		return "", fmt.Errorf("empty ban pattern")
	}
	if strings.Contains(p, "/") {
		_, ipnet, err := net.ParseCIDR(p)
		if err != nil {
			return "", fmt.Errorf("invalid CIDR %q: %w", p, err)
		}
		return ipnet.String(), nil
	}
	if ip := net.ParseIP(p); ip != nil {
		return ip.String(), nil
	}
	return "", fmt.Errorf("invalid IP or CIDR: %q", p)
}

// Add records a ban. The pattern is normalised; bannedBy is the SysOp's user id
// (0 for a system/CLI ban).
func (r *Bans) Add(pattern, reason string, bannedBy int64) (*Ban, error) {
	canon, err := NormalizeBanPattern(pattern)
	if err != nil {
		return nil, err
	}
	var by sql.NullInt64
	if bannedBy > 0 {
		by = sql.NullInt64{Int64: bannedBy, Valid: true}
	}
	res, err := r.st.db.Exec(
		`INSERT INTO ip_ban (pattern, reason, banned_by, banned_at) VALUES (?, ?, ?, ?)`,
		canon, strings.TrimSpace(reason), by, fmtTime(time.Now().UTC()))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.byID(id)
}

// Lift soft-removes a ban (sets lifted_at); the row is kept for history.
func (r *Bans) Lift(id int64) error {
	_, err := r.st.db.Exec(`UPDATE ip_ban SET lifted_at = ? WHERE id = ? AND lifted_at IS NULL`,
		fmtTime(time.Now().UTC()), id)
	return err
}

const banCols = `id, pattern, reason, banned_by, banned_at, lifted_at`

func (r *Bans) scan(row interface{ Scan(...any) error }) (*Ban, error) {
	var b Ban
	var bannedAt string
	var lifted sql.NullString
	if err := row.Scan(&b.ID, &b.Pattern, &b.Reason, &b.BannedBy, &bannedAt, &lifted); err != nil {
		return nil, err
	}
	b.BannedAt = parseTime(bannedAt)
	if lifted.Valid {
		t := parseTime(lifted.String)
		b.LiftedAt = &t
	}
	return &b, nil
}

func (r *Bans) byID(id int64) (*Ban, error) {
	return r.scan(r.st.db.QueryRow(`SELECT `+banCols+` FROM ip_ban WHERE id = ?`, id))
}

// Active returns all bans currently in force, newest first.
func (r *Bans) Active() ([]*Ban, error) {
	rows, err := r.st.db.Query(`SELECT ` + banCols + ` FROM ip_ban WHERE lifted_at IS NULL ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Ban
	for rows.Next() {
		b, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// IsBanned reports whether ip (a bare host, e.g. "203.0.113.7") is covered by an
// active ban — exact match or inside a banned CIDR. A malformed ip or empty
// banlist is simply not banned (fail-open at the transport edge is acceptable
// because real auth still follows; a DB error likewise must not wedge the door).
func (r *Bans) IsBanned(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	bans, err := r.Active()
	if err != nil {
		return false
	}
	for _, b := range bans {
		if strings.Contains(b.Pattern, "/") {
			if _, ipnet, err := net.ParseCIDR(b.Pattern); err == nil && ipnet.Contains(parsed) {
				return true
			}
			continue
		}
		if banned := net.ParseIP(b.Pattern); banned != nil && banned.Equal(parsed) {
			return true
		}
	}
	return false
}
