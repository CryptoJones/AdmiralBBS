package tests

import (
	"testing"
	"time"
)

func TestLoginAnomalyFlagging(t *testing.T) {
	s, _ := openTestStore(t)
	u, _ := s.Users().Create("alice", "", "", "")
	users := s.Users()

	base := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

	// First login establishes a baseline IP — never an anomaly.
	if flagged, err := users.RecordLogin(u.ID, "203.0.113.7", base); err != nil || flagged {
		t.Fatalf("first login: flagged=%v err=%v (want false/nil)", flagged, err)
	}

	// Same IP a minute later — not an anomaly.
	if flagged, _ := users.RecordLogin(u.ID, "203.0.113.7", base.Add(time.Minute)); flagged {
		t.Fatal("same IP should not flag")
	}

	// Different IP two minutes later — within the window → flagged.
	if flagged, _ := users.RecordLogin(u.ID, "198.51.100.4", base.Add(3*time.Minute)); !flagged {
		t.Fatal("rapid IP change should flag")
	}

	// Different IP, but well outside the window → not flagged (normal roaming).
	if flagged, _ := users.RecordLogin(u.ID, "192.0.2.9", base.Add(3*time.Hour)); flagged {
		t.Fatal("IP change outside the window should not flag")
	}

	rec, _ := s.Anomalies().Recent(10)
	if len(rec) != 1 {
		t.Fatalf("recorded anomalies = %d, want 1", len(rec))
	}
	a := rec[0]
	if a.PrevIP != "203.0.113.7" || a.NewIP != "198.51.100.4" || a.UserID != u.ID {
		t.Fatalf("anomaly fields wrong: %+v", a)
	}
	if n, _ := s.Anomalies().Count(); n != 1 {
		t.Fatalf("anomaly count = %d, want 1", n)
	}
}
