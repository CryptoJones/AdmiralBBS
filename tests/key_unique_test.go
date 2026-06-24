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

// A key may map to ONE sysop-tier account AND ONE regular account, but not two
// of the same tier — so an operator can keep a SysOp + a test user on one key.
func TestKeySharedAcrossTiers(t *testing.T) {
	s, _ := openTestStore(t)
	keys := s.Keys()
	line := genSSHKey(t)

	admin, _ := s.Users().Create("admin", "", "", "")
	s.Users().Approve(admin.ID, 100) // sysop tier
	tester, _ := s.Users().Create("tester", "", "", "")
	s.Users().Approve(tester.ID, 50) // regular tier

	if _, err := keys.Add(admin.ID, line); err != nil {
		t.Fatalf("sysop add: %v", err)
	}
	// Same key on a regular account is allowed (different tier).
	if _, err := keys.Add(tester.ID, line); err != nil {
		t.Fatalf("regular should be able to share the key with the sysop: %v", err)
	}

	// A second SYSOP on the same key is rejected (sysop slot taken).
	admin2, _ := s.Users().Create("admin2", "", "", "")
	s.Users().Approve(admin2.ID, 100)
	if _, err := keys.Add(admin2.ID, line); !errors.Is(err, store.ErrKeyTaken) {
		t.Fatalf("second sysop on the key: want ErrKeyTaken, got %v", err)
	}
	// A second REGULAR on the same key is rejected (regular slot taken).
	tester2, _ := s.Users().Create("tester2", "", "", "")
	s.Users().Approve(tester2.ID, 50)
	if _, err := keys.Add(tester2.ID, line); !errors.Is(err, store.ErrKeyTaken) {
		t.Fatalf("second regular on the key: want ErrKeyTaken, got %v", err)
	}
}
