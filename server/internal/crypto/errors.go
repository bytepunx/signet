package crypto

import "errors"

var (
	// ErrKeyNotSet is returned when an operation requires the master key but the
	// server is sealed and no key has been loaded into memory.
	ErrKeyNotSet = errors.New("master key not set: server is sealed")

	// ErrInvalidKeySize is returned when a key argument is not exactly KeySize bytes.
	// AES-256 requires a 32-byte key.
	ErrInvalidKeySize = errors.New("invalid key size: AES-256 requires a 32-byte key")

	// ErrInvalidCiphertext is returned when ciphertext is too short to contain a
	// valid nonce and GCM authentication tag.
	ErrInvalidCiphertext = errors.New("invalid ciphertext: data is too short or malformed")

	// ErrAuthenticationFailed is returned when GCM tag verification fails. This
	// indicates the ciphertext was tampered with or the wrong key was used.
	ErrAuthenticationFailed = errors.New("authentication failed: ciphertext is corrupted or the key is incorrect")
)
