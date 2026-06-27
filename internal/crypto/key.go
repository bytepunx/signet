package crypto

import (
	"crypto/rand"
	"fmt"
	"io"
	"sync"

	"github.com/awnumar/memguard"
)

// KeySize is the required key length in bytes for AES-256.
const KeySize = 32

// GenerateKey returns a cryptographically random 32-byte key suitable for AES-256.
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	return key, nil
}

// KeyStore holds the master key in mlock'd memory with guard pages and canaries.
// It is safe for concurrent use. The key is never returned directly; callers
// access it through Use, which scopes the lifetime of any reference.
//
// Set zeroes the caller's key slice after copying it — callers must not read
// the slice after calling Set.
type KeyStore struct {
	mu  sync.RWMutex
	buf *memguard.LockedBuffer
}

// NewKeyStore returns an empty, sealed KeyStore.
func NewKeyStore() *KeyStore {
	return &KeyStore{}
}

// Set loads key into locked memory and zeroes the caller's slice. Returns
// ErrInvalidKeySize if key is not exactly KeySize bytes.
func (s *KeyStore) Set(key []byte) error {
	if len(key) != KeySize {
		return ErrInvalidKeySize
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.buf != nil && s.buf.IsAlive() {
		s.buf.Destroy()
	}

	// NewBufferFromBytes copies the data into mlock'd memory and zeroes key.
	s.buf = memguard.NewBufferFromBytes(key)
	return nil
}

// IsSet reports whether the master key is currently loaded.
func (s *KeyStore) IsSet() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.buf != nil && s.buf.IsAlive()
}

// Use calls fn with the master key bytes. The byte slice is valid only for the
// duration of fn — callers must not retain a reference past the return. Returns
// ErrKeyNotSet if no key has been loaded.
func (s *KeyStore) Use(fn func([]byte) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.buf == nil || !s.buf.IsAlive() {
		return ErrKeyNotSet
	}

	return fn(s.buf.Bytes())
}

// Zero permanently wipes the master key from memory. After Zero returns,
// IsSet returns false and Use returns ErrKeyNotSet. Calling Zero on an already
// zeroed store is a no-op.
func (s *KeyStore) Zero() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.buf != nil && s.buf.IsAlive() {
		s.buf.Destroy()
	}
	s.buf = nil
}
