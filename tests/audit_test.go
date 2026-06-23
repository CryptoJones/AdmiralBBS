package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"admiralbbs/src/audit"
)

// The hash-chain must detect an edit to any line (SEC-6).
func TestAuditChainDetectsTampering(t *testing.T) {
	v := testVault(t)
	p := filepath.Join(t.TempDir(), "a.jsonl")
	lg, err := audit.New(p, v)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		lg.Emit(audit.Event{Type: audit.TypeActivity, SessionID: "s", Action: "a", Time: time.Now()})
	}
	lg.Close()

	// A clean trail verifies.
	if _, err := audit.ReadAll(p, v); err != nil {
		t.Fatalf("clean chain failed to verify: %v", err)
	}

	// Tamper with the first line's ciphertext (flip one base64 char).
	data, _ := os.ReadFile(p)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	b := []byte(lines[0])
	if b[0] == 'A' {
		b[0] = 'B'
	} else {
		b[0] = 'A'
	}
	lines[0] = string(b)
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := audit.ReadAll(p, v); err != audit.ErrChainBroken {
		t.Fatalf("tampering not detected: err = %v", err)
	}
}
