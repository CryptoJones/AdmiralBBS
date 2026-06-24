package tests

import (
	"path/filepath"
	"testing"
	"time"

	"admiralbbs/src/audit"
	"admiralbbs/src/store"
)

func TestSysOpUserManagement(t *testing.T) {
	s, _ := openTestStore(t)
	a, _ := s.Users().Create("alice", "", "", "")
	s.Users().Create("bob", "", "", "")

	all, _ := s.Users().All()
	if len(all) != 2 {
		t.Fatalf("All() = %d, want 2", len(all))
	}
	if err := s.Users().SetDailyMinutes(a.ID, 90); err != nil {
		t.Fatal(err)
	}
	if err := s.Users().SetStatus(a.ID, store.StatusSuspended, 0); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Users().ByID(a.ID)
	if got.DailyMinutes != 90 || got.Status != store.StatusSuspended {
		t.Fatalf("management not persisted: %+v", got)
	}
}

func TestSysOpApprovalIssuesUsableToken(t *testing.T) {
	s, _ := openTestStore(t)
	applicant, _ := s.Users().Create("newbie", "", "", "")
	sysop, _ := s.Users().Create("sysop", "", "", "")
	m, _ := s.Memberships().Apply(applicant.ID, "let me in")

	// Panel approve path: Approve + Review + Issue token.
	if err := s.Users().Approve(applicant.ID, 50); err != nil {
		t.Fatal(err)
	}
	s.Memberships().Review(m.ID, sysop.ID, store.DecisionApproved, "ok")
	tok, err := s.Tokens().Issue(applicant.ID)
	if err != nil || tok == "" {
		t.Fatalf("issue: %v", err)
	}
	got, _ := s.Users().ByID(applicant.ID)
	if got.Status != store.StatusApproved || got.AccessLevel != 50 {
		t.Fatalf("not approved: %+v", got)
	}
	// The issued token works for onboarding.
	if err := s.Tokens().Redeem(applicant.ID, tok); err != nil {
		t.Fatalf("issued token did not redeem: %v", err)
	}
}

func TestSysOpAuditViewerAndChainVerify(t *testing.T) {
	s, v := openTestStore(t)
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	lg, _ := audit.New(auditPath, v, s.SessionLog())
	lg.Emit(audit.Event{Type: audit.TypeActivity, SessionID: "s-1", Username: "alice", Action: "post-message", Detail: "secret detail", Time: time.Now()})
	lg.Emit(audit.Event{Type: audit.TypeDisconnect, SessionID: "s-1", Minutes: 2, Time: time.Now()})
	lg.Close()

	recent, err := s.SessionLog().Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 2 {
		t.Fatalf("recent = %d, want 2", len(recent))
	}
	// detail decrypts in the viewer
	var sawDetail bool
	for _, e := range recent {
		if e.Detail == "secret detail" {
			sawDetail = true
		}
	}
	if !sawDetail {
		t.Fatal("audit viewer did not decrypt detail")
	}
	// chain verifies intact
	n, err := s.VerifyAuditChain(auditPath)
	if err != nil || n != 2 {
		t.Fatalf("chain verify: n=%d err=%v", n, err)
	}
}
