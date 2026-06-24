// Package crypto provides AdmiralBBS's encryption-at-rest primitive: a Vault
// holding a symmetric key derived (Argon2id) from a startup secret that is
// never written to the data volume. Sensitive payloads are sealed with
// XChaCha20-Poly1305 (24-byte random nonces, AEAD).
//
// Threat model (see planning/RISKS.md): this makes everything on disk
// ciphertext — a stolen disk, copied volume, image layer, backup, or stopped
// container is unreadable without the key. It does NOT defend against an
// attacker with live root on the running host, who can scrape the key from
// process memory; fully closing that needs hardware (TPM/HSM/enclave). The key
// is mlock'd (never swapped to disk) and zeroed on Close to raise that bar.
package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/sys/unix"
)

// Argon2id KDF parameters for deriving the master key from the startup secret.
const (
	kdfTime    = 3
	kdfMemory  = 64 * 1024 // 64 MiB
	kdfThreads = 4
	keyLen     = chacha20poly1305.KeySize // 32
	saltLen    = 16
)

// ErrNoSecret is returned when no startup secret is supplied.
var ErrNoSecret = errors.New("no encryption secret supplied (set ADMIRALBBS_KEY)")

// Vault seals and opens data with a key held only in memory.
type Vault struct {
	aead interface {
		Seal(dst, nonce, plaintext, additionalData []byte) []byte
		Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error)
		NonceSize() int
	}
	key    []byte
	macKey []byte // domain-separated sub-key for the audit hash-chain
}

// NewVault derives the master key from secret + salt (Argon2id) and builds the
// AEAD. The key is mlock'd best-effort. Returns ErrNoSecret if secret is empty.
func NewVault(secret, salt []byte) (*Vault, error) {
	if len(secret) == 0 {
		return nil, ErrNoSecret
	}
	if len(salt) != saltLen {
		return nil, fmt.Errorf("salt must be %d bytes, got %d", saltLen, len(salt))
	}
	key := argon2.IDKey(secret, salt, kdfTime, kdfMemory, kdfThreads, keyLen)

	// Pin the key in RAM so it never swaps to disk. Best-effort: mlock can fail
	// under a low RLIMIT_MEMLOCK; that is not fatal, but we note it.
	if err := unix.Mlock(key); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not mlock key (%v); key may swap to disk\n", err)
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	// Derive a separate MAC key (key separation) for the audit hash-chain.
	mk := hmac.New(sha256.New, key)
	mk.Write([]byte("admiralbbs/audit-mac/v1"))
	macKey := mk.Sum(nil)

	return &Vault{aead: aead, key: key, macKey: macKey}, nil
}

// MAC returns HMAC-SHA256(macKey, prev || msg) — the link function for the
// tamper-evident audit hash-chain (each line commits to the previous line's
// MAC). Invented by Bellare, Canetti & Krawczyk (1996).
func (v *Vault) MAC(prev, msg []byte) []byte {
	h := hmac.New(sha256.New, v.macKey)
	h.Write(prev)
	h.Write(msg)
	return h.Sum(nil)
}

// Seal encrypts plaintext, returning nonce||ciphertext||tag.
func (v *Vault) Seal(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return v.aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Open decrypts a nonce||ciphertext||tag blob produced by Seal.
func (v *Vault) Open(blob []byte) ([]byte, error) {
	ns := v.aead.NonceSize()
	if len(blob) < ns {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := blob[:ns], blob[ns:]
	return v.aead.Open(nil, nonce, ct, nil)
}

// Close zeroes and unlocks the key material.
func (v *Vault) Close() {
	if v.key != nil {
		unix.Munlock(v.key)
		for i := range v.key {
			v.key[i] = 0
		}
		v.key = nil
	}
	for i := range v.macKey {
		v.macKey[i] = 0
	}
	v.macKey = nil
}

// GenerateSalt returns a fresh random KDF salt (for key rotation).
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// LoadOrCreateSalt reads the KDF salt from path, generating and persisting a
// random one (0600) on first run. The salt is not secret; it only needs to be
// stable per deployment so the same secret derives the same key.
func LoadOrCreateSalt(path string) ([]byte, error) {
	if data, err := os.ReadFile(path); err == nil {
		if len(data) != saltLen {
			return nil, fmt.Errorf("salt file %s is %d bytes, expected %d", path, len(data), saltLen)
		}
		return data, nil
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, salt, 0o600); err != nil {
		return nil, err
	}
	return salt, nil
}
