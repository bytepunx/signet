package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
)

// nonceSize is the standard GCM nonce length in bytes.
const nonceSize = 12

// AAD context tags. These are the fixed first argument to BindAAD for each
// class of encrypted artifact so producer and consumer sides always agree on
// the exact bytes, and so ciphertexts from different artifact classes can
// never be swapped for one another even if their other identifying fields
// happen to collide.
const (
	// AADSecret binds a secret's ciphertext (and its DEK's wrap under a KEK)
	// to BindAAD(AADSecret, namespace, service, name).
	AADSecret = "signet-secret"
	// AADKEK binds a KEK's wrap under the master key.
	AADKEK = "signet-kek"
	// AADKeyCheckValue binds the key-check value used to verify a candidate
	// master key immediately after unseal.
	AADKeyCheckValue = "signet-key-check-value"
	// AADRepoWebhookSecret binds a git repository's webhook HMAC secret to
	// BindAAD(AADRepoWebhookSecret, repoName). Not itself a credential — it's
	// a fixed public label, not the secret value.
	AADRepoWebhookSecret = "signet-repo-webhook-secret" //nolint:gosec // false positive: AAD label, not a credential
	// AADRepoDeployKey binds a git repository's SSH deploy key to
	// BindAAD(AADRepoDeployKey, repoName).
	AADRepoDeployKey = "signet-repo-deploy-key"
	// AADSOPSAgeKey binds a SOPS age private key to
	// BindAAD(AADSOPSAgeKey, publicKey).
	AADSOPSAgeKey = "signet-sops-age-key"
)

// BindAAD builds GCM additional authenticated data from a sequence of context
// parts (e.g. namespace, service, secret name), binding a ciphertext to the
// logical identity it protects so it cannot be silently swapped with another
// row's ciphertext by a party with database write access but no key material.
// Each part is length-prefixed to prevent ambiguity at part boundaries
// (e.g. ("ab","c") must not collide with ("a","bc")).
//
// BindAAD never returns nil/empty for a non-empty parts list, and returns nil
// for an empty list — callers that pass no parts get legacy nil-AAD behavior.
func BindAAD(parts ...string) []byte {
	if len(parts) == 0 {
		return nil
	}
	var buf []byte
	var lenPrefix [4]byte
	for _, p := range parts {
		binary.BigEndian.PutUint32(lenPrefix[:], uint32(len(p)))
		buf = append(buf, lenPrefix[:]...)
		buf = append(buf, p...)
	}
	return buf
}

// Encrypt encrypts plaintext with the given AES-256 key using AES-256-GCM.
// The returned ciphertext is formatted as: nonce (12 bytes) || ciphertext || GCM tag (16 bytes).
// aad is authenticated but not encrypted or stored; the same aad must be passed
// to Decrypt to recover the plaintext. Pass nil for no binding (legacy behavior).
// Returns ErrInvalidKeySize if key is not KeySize bytes.
func Encrypt(key, plaintext, aad []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("encrypt: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encrypt: create GCM: %w", err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("encrypt: generate nonce: %w", err)
	}

	// Seal appends the encrypted plaintext and GCM tag to nonce in one allocation.
	return gcm.Seal(nonce, nonce, plaintext, aad), nil
}

// Decrypt decrypts ciphertext produced by Encrypt. Expects the format:
// nonce (12 bytes) || ciphertext || GCM tag (16 bytes). aad must exactly match
// the value passed to Encrypt or authentication fails.
// Returns ErrInvalidKeySize, ErrInvalidCiphertext, or ErrAuthenticationFailed on failure.
func Decrypt(key, ciphertext, aad []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("decrypt: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("decrypt: create GCM: %w", err)
	}

	if len(ciphertext) < nonceSize+gcm.Overhead() {
		return nil, ErrInvalidCiphertext
	}

	nonce, body := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, body, aad)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuthenticationFailed, err)
	}

	return plaintext, nil
}

// DecryptWithFallback decrypts ciphertext using aad, and if that fails
// authentication, retries with nil AAD. This allows reading data written
// before AAD binding was introduced without a forced migration.
// legacy reports whether the nil-AAD fallback path was the one that succeeded,
// so callers can log a warning and/or trigger re-encryption under the current
// scheme. legacy is always false when err != nil.
func DecryptWithFallback(key, ciphertext, aad []byte) (plaintext []byte, legacy bool, err error) {
	plaintext, err = Decrypt(key, ciphertext, aad)
	if err == nil {
		return plaintext, false, nil
	}
	if len(aad) == 0 {
		return nil, false, err
	}
	plaintext, err = Decrypt(key, ciphertext, nil)
	if err != nil {
		return nil, false, err
	}
	return plaintext, true, nil
}

// WrapKey encrypts dek with kek using AES-256-GCM. Both arguments must be
// exactly KeySize bytes. aad binds the wrapped DEK to its logical identity;
// see Encrypt. Returns ErrInvalidKeySize if either key is the wrong length.
func WrapKey(kek, dek, aad []byte) ([]byte, error) {
	if len(kek) != KeySize {
		return nil, fmt.Errorf("wrap key: kek: %w", ErrInvalidKeySize)
	}
	if len(dek) != KeySize {
		return nil, fmt.Errorf("wrap key: dek: %w", ErrInvalidKeySize)
	}
	wrapped, err := Encrypt(kek, dek, aad)
	if err != nil {
		return nil, fmt.Errorf("wrap key: %w", err)
	}
	return wrapped, nil
}

// UnwrapKey decrypts a wrapped DEK produced by WrapKey. aad must match the
// value passed to WrapKey. Returns ErrInvalidKeySize if kek is the wrong
// length, ErrInvalidCiphertext if wrapped is malformed, or
// ErrAuthenticationFailed if the kek is incorrect, aad does not match, or the
// wrapped data is tampered.
func UnwrapKey(kek, wrapped, aad []byte) ([]byte, error) {
	if len(kek) != KeySize {
		return nil, fmt.Errorf("unwrap key: kek: %w", ErrInvalidKeySize)
	}
	dek, err := Decrypt(kek, wrapped, aad)
	if err != nil {
		return nil, fmt.Errorf("unwrap key: %w", err)
	}
	if len(dek) != KeySize {
		return nil, fmt.Errorf("unwrap key: decrypted payload is %d bytes, expected %d", len(dek), KeySize)
	}
	return dek, nil
}

// UnwrapKeyWithFallback is UnwrapKey with the same legacy nil-AAD fallback
// behavior as DecryptWithFallback.
func UnwrapKeyWithFallback(kek, wrapped, aad []byte) (dek []byte, legacy bool, err error) {
	if len(kek) != KeySize {
		return nil, false, fmt.Errorf("unwrap key: kek: %w", ErrInvalidKeySize)
	}
	dek, legacy, err = DecryptWithFallback(kek, wrapped, aad)
	if err != nil {
		return nil, false, fmt.Errorf("unwrap key: %w", err)
	}
	if len(dek) != KeySize {
		return nil, false, fmt.Errorf("unwrap key: decrypted payload is %d bytes, expected %d", len(dek), KeySize)
	}
	return dek, legacy, nil
}
