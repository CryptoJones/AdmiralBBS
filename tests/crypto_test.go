package tests

import (
	"bytes"
	"testing"

	"admiralbbs/src/crypto"
)

func TestVaultSealOpenRoundTrip(t *testing.T) {
	v := testVault(t)
	ct, err := v.Seal([]byte("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ct, []byte("hello world")) {
		t.Fatal("ciphertext contains the plaintext")
	}
	pt, err := v.Open(ct)
	if err != nil {
		t.Fatal(err)
	}
	if string(pt) != "hello world" {
		t.Fatalf("round trip = %q", pt)
	}
}

func TestVaultWrongKeyFails(t *testing.T) {
	v := testVault(t)
	ct, _ := v.Seal([]byte("secret"))
	other, err := crypto.NewVault([]byte("a-different-secret"), []byte("0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if _, err := other.Open(ct); err == nil {
		t.Fatal("a different key must not open the ciphertext")
	}
}

func TestVaultMACChaining(t *testing.T) {
	v := testVault(t)
	a := v.MAC(nil, []byte("x"))
	b := v.MAC(nil, []byte("x"))
	if !bytes.Equal(a, b) {
		t.Fatal("MAC is not deterministic")
	}
	chained := v.MAC(a, []byte("x"))
	if bytes.Equal(a, chained) {
		t.Fatal("chaining on prev MAC had no effect")
	}
}
