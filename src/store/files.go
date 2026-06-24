package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MaxFileBytes caps a single upload; MaxUserBytes caps a user's total stored
// bytes — both blunt disk-exhaustion (RISKS SEC-7).
const (
	MaxFileBytes = 10 << 20  // 10 MiB per file
	MaxUserBytes = 100 << 20 // 100 MiB total per uploader
)

// ErrTooLarge is returned when an upload exceeds MaxFileBytes.
var ErrTooLarge = errors.New("file exceeds the size limit")

// ErrQuotaExceeded is returned when an upload would exceed the user's quota.
var ErrQuotaExceeded = errors.New("upload would exceed your storage quota")

// ErrDuplicateName is returned when a file of that name already exists in the area.
var ErrDuplicateName = errors.New("a file with that name already exists in this area")

// FileArea is a download area in the file library.
type FileArea struct {
	ID             int64
	Name           string
	MinAccessLevel int
}

// FileEntry is a downloadable object. The blob lives sealed on disk; only
// metadata is in the DB. The on-disk path is derived from the row id, so a
// hostile filename can never traverse out of the files dir (SEC-7).
type FileEntry struct {
	ID            int64
	AreaID        int64
	Filename      string // display name only
	SizeBytes     int64
	Description   string
	UploaderID    int64
	DownloadCount int64
	UploadedAt    time.Time
}

// FileAreas is the file-area repository.
type FileAreas struct{ st *Store }

func (r *FileAreas) Create(name string, minLevel int) (*FileArea, error) {
	res, err := r.st.db.Exec(`INSERT INTO file_area (name, min_access_level) VALUES (?, ?)`, name, minLevel)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &FileArea{ID: id, Name: name, MinAccessLevel: minLevel}, nil
}

func (r *FileAreas) Count() (int, error) {
	var n int
	err := r.st.db.QueryRow(`SELECT COUNT(*) FROM file_area`).Scan(&n)
	return n, err
}

func (r *FileAreas) Visible(accessLevel int) ([]*FileArea, error) {
	rows, err := r.st.db.Query(`SELECT id, name, min_access_level FROM file_area WHERE min_access_level <= ? ORDER BY name`, accessLevel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*FileArea
	for rows.Next() {
		var a FileArea
		if err := rows.Scan(&a.ID, &a.Name, &a.MinAccessLevel); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *FileAreas) ByID(id int64, accessLevel int) (*FileArea, error) {
	var a FileArea
	err := r.st.db.QueryRow(`SELECT id, name, min_access_level FROM file_area WHERE id = ?`, id).
		Scan(&a.ID, &a.Name, &a.MinAccessLevel)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && accessLevel < a.MinAccessLevel) {
		return nil, ErrNotFound
	}
	return &a, err
}

// Files is the file-entry repository.
type Files struct{ st *Store }

func (r *Files) blobPath(id int64) string {
	// id-based name — never derived from the caller's filename (SEC-7).
	return filepath.Join(r.st.filesDir, fmt.Sprintf("%d.bin", id))
}

// Add stores an uploaded blob: the row is inserted first (to allocate an id),
// then the content is sealed and written to <id>.bin. Returns ErrTooLarge if
// the content exceeds MaxFileBytes.
func (r *Files) Add(areaID, uploaderID int64, filename, description string, content []byte) (*FileEntry, error) {
	if int64(len(content)) > MaxFileBytes {
		return nil, ErrTooLarge
	}
	// Serialize uploads so the quota check + insert is atomic — otherwise two
	// concurrent uploads by one user both pass the check and blow the quota.
	r.st.uploadMu.Lock()
	defer r.st.uploadMu.Unlock()
	// Reject a duplicate filename in the same area (case-insensitive) — covers
	// re-uploading the same file; delete the old one to replace it.
	var dup int
	if err := r.st.db.QueryRow(`SELECT COUNT(*) FROM file_entry WHERE area_id = ? AND filename = ? COLLATE NOCASE`, areaID, filename).Scan(&dup); err != nil {
		return nil, err
	}
	if dup > 0 {
		return nil, ErrDuplicateName
	}
	used, err := r.UserBytes(uploaderID)
	if err != nil {
		return nil, err
	}
	if used+int64(len(content)) > MaxUserBytes {
		return nil, ErrQuotaExceeded
	}
	now := time.Now().UTC()
	res, err := r.st.db.Exec(
		`INSERT INTO file_entry (area_id, filename, path, size_bytes, description, uploader_id, uploaded_at)
		 VALUES (?, ?, '', ?, ?, ?, ?)`,
		areaID, filename, len(content), description, uploaderID, fmtTime(now))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	sealed, err := r.st.vault.Seal(content)
	if err != nil {
		return nil, err
	}
	path := r.blobPath(id)
	if err := os.WriteFile(path, sealed, 0o600); err != nil {
		return nil, err
	}
	if _, err := r.st.db.Exec(`UPDATE file_entry SET path = ? WHERE id = ?`, filepath.Base(path), id); err != nil {
		return nil, err
	}
	return &FileEntry{ID: id, AreaID: areaID, Filename: filename, SizeBytes: int64(len(content)),
		Description: description, UploaderID: uploaderID, UploadedAt: now}, nil
}

const fileCols = `id, area_id, filename, size_bytes, description, uploader_id, download_count, uploaded_at`

func (r *Files) scan(row interface{ Scan(...any) error }) (*FileEntry, error) {
	var f FileEntry
	var uploaded string
	if err := row.Scan(&f.ID, &f.AreaID, &f.Filename, &f.SizeBytes, &f.Description, &f.UploaderID, &f.DownloadCount, &uploaded); err != nil {
		return nil, err
	}
	f.UploadedAt = parseTime(uploaded)
	return &f, nil
}

// ListByArea lists files in an area, newest first.
func (r *Files) ListByArea(areaID int64) ([]*FileEntry, error) {
	rows, err := r.st.db.Query(`SELECT `+fileCols+` FROM file_entry WHERE area_id = ? ORDER BY id DESC`, areaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*FileEntry
	for rows.Next() {
		f, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// UserBytes returns the total bytes a user has uploaded (for the quota check).
func (r *Files) UserBytes(uploaderID int64) (int64, error) {
	var total sql.NullInt64
	err := r.st.db.QueryRow(`SELECT COALESCE(SUM(size_bytes), 0) FROM file_entry WHERE uploader_id = ?`, uploaderID).Scan(&total)
	return total.Int64, err
}

// ListSorted lists files by name, date, size, or downloads. Filenames and
// descriptions are NOT encrypted, so these are direct SQL (unlike board search).
func (r *Files) ListSorted(areaID int64, by string) ([]*FileEntry, error) {
	order := "id DESC"
	switch by {
	case "name":
		order = "filename COLLATE NOCASE ASC"
	case "size":
		order = "size_bytes DESC"
	case "downloads":
		order = "download_count DESC"
	case "oldest":
		order = "id ASC"
	}
	return r.listFiles(`SELECT `+fileCols+` FROM file_entry WHERE area_id = ? ORDER BY `+order, areaID)
}

// Search returns files whose filename or description matches query (SQL LIKE,
// case-insensitive — these fields are plaintext).
func (r *Files) Search(areaID int64, query string) ([]*FileEntry, error) {
	like := "%" + query + "%"
	return r.listFiles(`SELECT `+fileCols+` FROM file_entry WHERE area_id = ? AND (filename LIKE ? OR description LIKE ?) ORDER BY id DESC`, areaID, like, like)
}

// ByUploader lists an area's files from one uploader.
func (r *Files) ByUploader(areaID, uploaderID int64) ([]*FileEntry, error) {
	return r.listFiles(`SELECT `+fileCols+` FROM file_entry WHERE area_id = ? AND uploader_id = ? ORDER BY id DESC`, areaID, uploaderID)
}

func (r *Files) listFiles(query string, args ...any) ([]*FileEntry, error) {
	rows, err := r.st.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*FileEntry
	for rows.Next() {
		f, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// Delete removes a file's row and its blob. Only the uploader or a SysOp should
// be allowed to call this (enforced by the menu).
func (r *Files) Delete(id int64) error {
	if _, err := r.st.db.Exec(`DELETE FROM file_entry WHERE id = ?`, id); err != nil {
		return err
	}
	return os.Remove(r.blobPath(id))
}

// ByID fetches one file's metadata.
func (r *Files) ByID(id int64) (*FileEntry, error) {
	row := r.st.db.QueryRow(`SELECT `+fileCols+` FROM file_entry WHERE id = ?`, id)
	f, err := r.scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return f, err
}

// Content decrypts and returns a file's bytes, and bumps the download counter.
func (r *Files) Content(id int64) ([]byte, error) {
	sealed, err := os.ReadFile(r.blobPath(id))
	if err != nil {
		return nil, err
	}
	plain, err := r.st.vault.Open(sealed)
	if err != nil {
		return nil, err
	}
	_, _ = r.st.db.Exec(`UPDATE file_entry SET download_count = download_count + 1 WHERE id = ?`, id)
	return plain, nil
}

// FileAreas returns the file-area repository.
func (s *Store) FileAreas() *FileAreas { return &FileAreas{st: s} }

// Files returns the file-entry repository.
func (s *Store) Files() *Files { return &Files{st: s} }

// EnsureSeedFileAreas creates a default download area on first run.
func (s *Store) EnsureSeedFileAreas() error {
	n, err := s.FileAreas().Count()
	if err != nil || n > 0 {
		return err
	}
	_, err = s.FileAreas().Create("General Files", 0)
	return err
}
