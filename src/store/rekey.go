package store

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"admiralbbs/src/crypto"
	_ "modernc.org/sqlite"
)

// sealedColumns lists every base64-RawStd-sealed TEXT column, by table.
var sealedColumns = []struct {
	table string
	cols  []string
}{
	{"user", []string{"real_name", "email"}},
	{"membership", []string{"note"}},
	{"message", []string{"subject", "body"}},
	{"private_message", []string{"subject", "body"}},
	{"session_log", []string{"detail"}},
}

// RekeyDB re-encrypts every sealed DB column and every file-library blob from
// oldV to newV, in one transaction for the DB. The database must NOT be open
// elsewhere. Part of the key-rotation runbook (docs/OPERATIONS.md).
func RekeyDB(dbPath string, oldV, newV *crypto.Vault) error {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(0)")
	if err != nil {
		return err
	}
	defer db.Close()

	enc := base64.RawStdEncoding
	reseal := func(s string) (string, error) {
		if s == "" {
			return "", nil
		}
		ct, err := enc.DecodeString(s)
		if err != nil {
			return "", err
		}
		plain, err := oldV.Open(ct)
		if err != nil {
			return "", fmt.Errorf("decrypt with old key: %w", err)
		}
		nct, err := newV.Seal(plain)
		if err != nil {
			return "", err
		}
		return enc.EncodeToString(nct), nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, t := range sealedColumns {
		if err := rekeyTable(tx, t.table, t.cols, reseal); err != nil {
			tx.Rollback()
			return fmt.Errorf("rekey %s: %w", t.table, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	// File-library blobs: <db-dir>/files/*.bin
	filesDir := filepath.Join(filepath.Dir(dbPath), "files")
	entries, err := os.ReadDir(filesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".bin" {
			continue
		}
		p := filepath.Join(filesDir, e.Name())
		sealed, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		plain, err := oldV.Open(sealed)
		if err != nil {
			return fmt.Errorf("rekey blob %s: decrypt with old key: %w", e.Name(), err)
		}
		nct, err := newV.Seal(plain)
		if err != nil {
			return err
		}
		if err := os.WriteFile(p, nct, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func rekeyTable(tx *sql.Tx, table string, cols []string, reseal func(string) (string, error)) error {
	sel := "SELECT id"
	for _, c := range cols {
		sel += ", " + c
	}
	sel += " FROM " + table
	rows, err := tx.Query(sel)
	if err != nil {
		return err
	}
	type rowVals struct {
		id   int64
		vals []string
	}
	var batch []rowVals
	for rows.Next() {
		vals := make([]string, len(cols))
		dest := make([]any, len(cols)+1)
		var id int64
		dest[0] = &id
		for i := range cols {
			dest[i+1] = &vals[i]
		}
		if err := rows.Scan(dest...); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, rowVals{id, append([]string(nil), vals...)})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	set := ""
	for i, c := range cols {
		if i > 0 {
			set += ", "
		}
		set += c + " = ?"
	}
	upd := "UPDATE " + table + " SET " + set + " WHERE id = ?"
	for _, r := range batch {
		args := make([]any, 0, len(cols)+1)
		for _, v := range r.vals {
			nv, err := reseal(v)
			if err != nil {
				return err
			}
			args = append(args, nv)
		}
		args = append(args, r.id)
		if _, err := tx.Exec(upd, args...); err != nil {
			return err
		}
	}
	return nil
}
