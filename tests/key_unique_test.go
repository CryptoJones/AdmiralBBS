package tests

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"admiralbbs/src/store"
)

// One account per key: a fingerprint already active on one account cannot be
// registered on another (anti-sockpuppet).
func TestKeyFingerprintUniqueAcrossAccounts(t *testing.T) {
	s, _ := openTestStore(t)
	alice, _ := s.Users().Create("alice", "", "", "")
	bob, _ := s.Users().Create("bob", "", "", "")
	keys := s.Keys()

	line := genSSHKey(t)
	if _, err := keys.Add(alice.ID, line); err != nil {
		t.Fatalf("alice add: %v", err)
	}
	if _, err := keys.Add(bob.ID, line); !errors.Is(err, store.ErrKeyTaken) {
		t.Fatalf("bob reusing alice's key: want ErrKeyTaken, got %v", err)
	}

	// ByFingerprint resolves the key to its single owner.
	k, _ := keys.Active(alice.ID)
	owner, err := keys.ByFingerprint(k[0].Fingerprint)
	if err != nil || owner == nil || owner.UserID != alice.ID {
		t.Fatalf("ByFingerprint: want alice, got %+v (err %v)", owner, err)
	}
}

// Revoking a key frees its fingerprint: the same key may then be registered by
// someone else (a key that legitimately changes hands isn't blocked forever).
func TestKeyFingerprintFreedOnRevoke(t *testing.T) {
	s, _ := openTestStore(t)
	alice, _ := s.Users().Create("alice", "", "", "")
	bob, _ := s.Users().Create("bob", "", "", "")
	keys := s.Keys()

	line := genSSHKey(t)
	ak, _ := keys.Add(alice.ID, line)
	if err := keys.Revoke(ak.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := keys.Add(bob.ID, line); err != nil {
		t.Fatalf("bob adding alice's revoked key: %v", err)
	}
	if owner, _ := keys.ByFingerprint(ak.Fingerprint); owner == nil || owner.UserID != bob.ID {
		t.Fatalf("freed fingerprint should now resolve to bob, got %+v", owner)
	}
}

// The uniqueness guarantee is race-safe (DB constraint, not check-then-insert):
// many goroutines racing to claim the same key on different accounts yield
// exactly one winner.
func TestKeyFingerprintUniqueUnderRace(t *testing.T) {
	s, _ := openTestStore(t)
	keys := s.Keys()
	line := genSSHKey(t)
	users := make([]int64, 20)
	for i := range users {
		u, _ := s.Users().Create(handleN(i), "", "", "")
		users[i] = u.ID
	}
	var won int32
	var wg sync.WaitGroup
	for _, uid := range users {
		wg.Add(1)
		go func(uid int64) {
			defer wg.Done()
			if _, err := keys.Add(uid, line); err == nil {
				atomic.AddInt32(&won, 1)
			}
		}(uid)
	}
	wg.Wait()
	if won != 1 {
		t.Fatalf("race: %d accounts claimed the same key, want exactly 1", won)
	}
}

func handleN(i int) string { return "user" + string(rune('a'+i)) }
