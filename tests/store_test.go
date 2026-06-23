package tests

import (
	"path/filepath"
	"testing"
	"time"

	"admiralbbs/src/audit"
	"admiralbbs/src/crypto"
	"admiralbbs/src/store"
)

func openTestStore(t *testing.T) (*store.Store, *crypto.Vault) {
	t.Helper()
	v := testVault(t)
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(path, v)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s, v
}

func TestMigrateSetsUserVersionAndIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(path, testVault(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	var ver int
	if err := s.DB().QueryRow("PRAGMA user_version").Scan(&ver); err != nil {
		t.Fatal(err)
	}
	if ver != 2 {
		t.Fatalf("user_version = %d, want 2 (001_init + 002_approval_tokens)", ver)
	}
	var mode string
	if err := s.DB().QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
	s.Close()

	// Re-open: migrations must not re-run or error.
	s2, err := store.Open(path, testVault(t))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	if _, err := s2.Users().Create("reopen", "h", "", ""); err != nil {
		t.Fatalf("usable after reopen: %v", err)
	}
}

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := store.HashPassword("hunter2")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "hunter2" || len(hash) < 20 {
		t.Fatalf("hash looks wrong: %q", hash)
	}
	ok, err := store.VerifyPassword(hash, "hunter2")
	if err != nil || !ok {
		t.Fatalf("correct password rejected: ok=%v err=%v", ok, err)
	}
	bad, err := store.VerifyPassword(hash, "wrong")
	if err != nil {
		t.Fatal(err)
	}
	if bad {
		t.Fatal("wrong password accepted")
	}
}

func TestUserCreateAndLookup(t *testing.T) {
	s, _ := openTestStore(t)
	users := s.Users()

	hash, _ := store.HashPassword("pw")
	u, err := users.Create("SysOp", hash, "Aaron", "a@example.com")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.ID == 0 || u.Status != store.StatusPending {
		t.Fatalf("unexpected new user: %+v", u)
	}

	got, err := users.ByHandle("sysop") // case-insensitive
	if err != nil {
		t.Fatalf("by handle: %v", err)
	}
	if got.ID != u.ID || got.RealName != "Aaron" || got.Email != "a@example.com" {
		t.Fatalf("lookup/decrypt failed: %+v", got)
	}

	if _, err := users.Create("sysop", hash, "", ""); err == nil {
		t.Fatal("expected unique-handle violation, got nil")
	}

	if err := users.SetStatus(u.ID, store.StatusApproved, 100); err != nil {
		t.Fatal(err)
	}
	if err := users.TouchLogin(u.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	got, _ = users.ByID(u.ID)
	if got.Status != store.StatusApproved || got.AccessLevel != 100 || got.LastLoginAt == nil {
		t.Fatalf("approval/login not persisted: %+v", got)
	}

	if _, err := users.ByHandle("nobody"); err != store.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestMembershipWorkflow(t *testing.T) {
	s, _ := openTestStore(t)
	hash, _ := store.HashPassword("pw")
	applicant, _ := s.Users().Create("newbie", hash, "", "")
	sysop, _ := s.Users().Create("sysop", hash, "", "")

	m, err := s.Memberships().Apply(applicant.ID, "I love retro computing")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if m.Decision != store.DecisionPending || m.Note != "I love retro computing" {
		t.Fatalf("new application wrong: %+v", m)
	}

	if pending, _ := s.Memberships().Pending(); len(pending) != 1 {
		t.Fatalf("want 1 pending, got %d", len(pending))
	}

	if err := s.Memberships().Review(m.ID, sysop.ID, store.DecisionApproved, "welcome"); err != nil {
		t.Fatal(err)
	}
	reviewed, _ := s.Memberships().ByID(m.ID)
	if reviewed.Decision != store.DecisionApproved || reviewed.ReviewedBy == nil || *reviewed.ReviewedBy != sysop.ID || reviewed.Note != "welcome" {
		t.Fatalf("review not persisted: %+v", reviewed)
	}
	if pend, _ := s.Memberships().Pending(); len(pend) != 0 {
		t.Fatalf("approved app still pending: %d", len(pend))
	}
}

func TestAuditDualWriteMirrorsToSessionLog(t *testing.T) {
	s, v := openTestStore(t)
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	logger, err := audit.New(auditPath, v, s.SessionLog())
	if err != nil {
		t.Fatal(err)
	}

	logger.Emit(audit.Event{Type: audit.TypeConnect, SessionID: "s-1", RemoteIP: "203.0.113.7", Transport: "ssh", Username: "sysop", Time: time.Now()})
	logger.Emit(audit.Event{Type: audit.TypeActivity, SessionID: "s-1", Action: "message-boards", Detail: "top secret", Time: time.Now()})
	logger.Emit(audit.Event{Type: audit.TypeDisconnect, SessionID: "s-1", Minutes: 1.5, Time: time.Now()})
	logger.Close()

	n, err := s.SessionLog().Count()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("session_log mirror has %d rows, want 3", n)
	}

	// Structural columns stay queryable; free-text detail is encrypted.
	var ip, detail string
	if err := s.DB().QueryRow(`SELECT remote_ip FROM session_log WHERE event_type = ?`, audit.TypeConnect).Scan(&ip); err != nil {
		t.Fatal(err)
	}
	if ip != "203.0.113.7" {
		t.Fatalf("mirrored remote_ip = %q", ip)
	}
	if err := s.DB().QueryRow(`SELECT detail FROM session_log WHERE action = ?`, "message-boards").Scan(&detail); err != nil {
		t.Fatal(err)
	}
	if detail == "" || detail == "top secret" {
		t.Fatalf("detail should be encrypted at rest, got %q", detail)
	}
}
