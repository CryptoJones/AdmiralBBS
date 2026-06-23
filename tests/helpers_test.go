package tests

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"admiralbbs/src/crypto"
	"golang.org/x/crypto/ssh"
)

// testVault builds a deterministic vault (fixed secret+salt) for tests.
func testVault(t *testing.T) *crypto.Vault {
	t.Helper()
	v, err := crypto.NewVault([]byte("test-secret-please-ignore"), []byte("0123456789abcdef"))
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	t.Cleanup(v.Close)
	return v
}

// genSSHKey returns a fresh, valid authorized_keys line for tests.
func genSSHKey(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return string(ssh.MarshalAuthorizedKey(sshPub))
}
