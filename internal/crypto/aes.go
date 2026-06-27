package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// nonceSize is the standard GCM nonce length in bytes.
const nonceSize = 12

// Encrypt encrypts plaintext with the given AES-256 key using AES-256-GCM.
// The returned ciphertext is formatted as: nonce (12 bytes) || ciphertext || GCM tag (16 bytes).
// Returns ErrInvalidKeySize if key is not KeySize bytes.
func Encrypt(key, plaintext []byte) ([]byte, error) {
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
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext produced by Encrypt. Expects the format:
// nonce (12 bytes) || ciphertext || GCM tag (16 bytes).
// Returns ErrInvalidKeySize, ErrInvalidCiphertext, or ErrAuthenticationFailed on failure.
func Decrypt(key, ciphertext []byte) ([]byte, error) {
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

	plaintext, err := gcm.Open(nil, nonce, body, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuthenticationFailed, err)
	}

	return plaintext, nil
}

// WrapKey encrypts dek with kek using AES-256-GCM. Both arguments must be
// exactly KeySize bytes. Returns ErrInvalidKeySize if either is the wrong length.
func WrapKey(kek, dek []byte) ([]byte, error) {
	if len(kek) != KeySize {
		return nil, fmt.Errorf("wrap key: kek: %w", ErrInvalidKeySize)
	}
	if len(dek) != KeySize {
		return nil, fmt.Errorf("wrap key: dek: %w", ErrInvalidKeySize)
	}
	wrapped, err := Encrypt(kek, dek)
	if err != nil {
		return nil, fmt.Errorf("wrap key: %w", err)
	}
	return wrapped, nil
}

// UnwrapKey decrypts a wrapped DEK produced by WrapKey. Returns ErrInvalidKeySize
// if kek is the wrong length, ErrInvalidCiphertext if wrapped is malformed, or
// ErrAuthenticationFailed if the kek is incorrect or the wrapped data is tampered.
func UnwrapKey(kek, wrapped []byte) ([]byte, error) {
	if len(kek) != KeySize {
		return nil, fmt.Errorf("unwrap key: kek: %w", ErrInvalidKeySize)
	}
	dek, err := Decrypt(kek, wrapped)
	if err != nil {
		return nil, fmt.Errorf("unwrap key: %w", err)
	}
	if len(dek) != KeySize {
		return nil, fmt.Errorf("unwrap key: decrypted payload is %d bytes, expected %d", len(dek), KeySize)
	}
	return dek, nil
}
