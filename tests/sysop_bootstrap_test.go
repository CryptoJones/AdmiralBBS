package tests

import (
	"testing"

	"admiralbbs/src/store"
	"golang.org/x/crypto/ssh"
)

// A SysOp created the way `sysopctl bootstrap` creates one (handle + key +
// password + approve) must satisfy BOTH auth factors and reach the control
// panel — i.e. they can log in with no onboarding token.
func TestSysOpBootstrapAccountIsLoginReady(t *testing.T) {
	s, _ := openTestStore(t)
	keyLine := genSSHKey(t)

	// Mirror sysopctl bootstrap's store sequence.
	u, err := s.Users().Create("CryptoJones", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	hash, err := store.HashPassword("b000bies")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Users().SetPassword(u.ID, hash); err != nil {
		t.Fatal(err)
	}
	if err := s.Users().Approve(u.ID, 100); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Keys().Add(u.ID, keyLine); err != nil {
		t.Fatal(err)
	}

	// Reload and assert the account is approved at the SysOp level with a password.
	got, err := s.Users().ByHandle("CryptoJones")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.StatusApproved || got.AccessLevel != 100 || got.PasswordHash == "" {
		t.Fatalf("bootstrapped account not login-ready: %+v", got)
	}

	// Factor 1: the registered key authorizes.
	pub, _, _, _, perr := ssh.ParseAuthorizedKey([]byte(keyLine))
	if perr != nil {
		t.Fatal(perr)
	}
	if ok, _ := s.Keys().Authorizes(got.ID, pub); !ok {
		t.Fatal("registered SSH key should authorize")
	}
	// Factor 2: the password verifies.
	if ok, _ := store.VerifyPassword(got.PasswordHash, "b000bies"); !ok {
		t.Fatal("password should verify")
	}
}
