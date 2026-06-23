package tests

import (
	"testing"

	"admiralbbs/src/store"
	"golang.org/x/crypto/ssh"
)

func TestKeysAddActiveRevokeAuthorize(t *testing.T) {
	s, _ := openTestStore(t)
	u, _ := s.Users().Create("alice", "", "", "")
	keys := s.Keys()

	line1 := genSSHKey(t)
	line2 := genSSHKey(t)
	if _, err := keys.Add(u.ID, line1); err != nil {
		t.Fatalf("add key1: %v", err)
	}
	k2, err := keys.Add(u.ID, line2)
	if err != nil {
		t.Fatalf("add key2: %v", err)
	}

	if act, _ := keys.Active(u.ID); len(act) != 2 {
		t.Fatalf("active = %d, want 2", len(act))
	}

	// Soft revoke keeps the row but drops it from Active.
	if err := keys.Revoke(k2.ID); err != nil {
		t.Fatal(err)
	}
	if act, _ := keys.Active(u.ID); len(act) != 1 {
		t.Fatalf("active after revoke = %d, want 1", len(act))
	}
	if all, _ := keys.All(u.ID); len(all) != 2 {
		t.Fatalf("all = %d, want 2 (revoked kept)", len(all))
	}

	if err := store.ValidatePublicKey("not a public key"); err == nil {
		t.Fatal("junk key was accepted")
	}

	// Two-factor SSH layer: an active key authorizes; a foreign key does not.
	pub, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(line1))
	if ok, _ := keys.Authorizes(u.ID, pub); !ok {
		t.Fatal("active key was not authorized")
	}
	foreign, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(genSSHKey(t)))
	if ok, _ := keys.Authorizes(u.ID, foreign); ok {
		t.Fatal("foreign key was authorized")
	}
}
