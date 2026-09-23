package crypto

import (
	"bytes"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateKey covers key generation correctness and randomness.
func TestGenerateKey(t *testing.T) {
	t.Run("produces KeySize bytes", func(t *testing.T) {
		key, err := GenerateKey()
		require.NoError(t, err)
		assert.Len(t, key, KeySize)
	})

	t.Run("two calls produce different keys", func(t *testing.T) {
		a, err := GenerateKey()
		require.NoError(t, err)
		b, err := GenerateKey()
		require.NoError(t, err)
		assert.NotEqual(t, a, b)
	})
}

// TestKeyStore_Set covers loading a key into locked memory.
func TestKeyStore_Set(t *testing.T) {
	t.Run("valid key is accepted", func(t *testing.T) {
		s := NewKeyStore()
		key := mustGenerateKey(t)
		require.NoError(t, s.Set(key))
		assert.True(t, s.IsSet())
	})

	t.Run("zeroes the caller slice after copying", func(t *testing.T) {
		s := NewKeyStore()
		key := mustGenerateKey(t)
		require.NoError(t, s.Set(key))
		assert.Equal(t, make([]byte, KeySize), key, "input slice must be zeroed after Set")
	})

	t.Run("nil key returns ErrInvalidKeySize", func(t *testing.T) {
		s := NewKeyStore()
		err := s.Set(nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("short key returns ErrInvalidKeySize", func(t *testing.T) {
		s := NewKeyStore()
		err := s.Set(make([]byte, KeySize-1))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("long key returns ErrInvalidKeySize", func(t *testing.T) {
		s := NewKeyStore()
		err := s.Set(make([]byte, KeySize+1))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("empty key returns ErrInvalidKeySize", func(t *testing.T) {
		s := NewKeyStore()
		err := s.Set([]byte{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("replaces an existing key without error", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))
		require.NoError(t, s.Set(mustGenerateKey(t)))
		assert.True(t, s.IsSet())
	})
}

// TestKeyStore_IsSet covers sealed/unsealed state reporting.
func TestKeyStore_IsSet(t *testing.T) {
	t.Run("false on a new store", func(t *testing.T) {
		assert.False(t, NewKeyStore().IsSet())
	})

	t.Run("true after a successful Set", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))
		assert.True(t, s.IsSet())
	})

	t.Run("false after Zero", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))
		s.Zero()
		assert.False(t, s.IsSet())
	})
}

// TestKeyStore_Use covers key access scoping and error propagation.
func TestKeyStore_Use(t *testing.T) {
	t.Run("provides correct key bytes to fn", func(t *testing.T) {
		s := NewKeyStore()
		original := mustGenerateKey(t)
		expected := make([]byte, KeySize)
		copy(expected, original)

		require.NoError(t, s.Set(original))

		var got []byte
		err := s.Use(func(k []byte) error {
			got = make([]byte, len(k))
			copy(got, k)
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("fn receives key of length KeySize", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))
		err := s.Use(func(k []byte) error {
			assert.Len(t, k, KeySize)
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("returns ErrKeyNotSet when store is empty", func(t *testing.T) {
		err := NewKeyStore().Use(func(_ []byte) error { return nil })
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrKeyNotSet)
	})

	t.Run("propagates fn error to caller", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))
		sentinel := errors.New("fn failed")
		err := s.Use(func(_ []byte) error { return sentinel })
		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("key can be used for encryption inside fn", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))

		plaintext := []byte("hello signet")
		var ciphertext []byte

		err := s.Use(func(k []byte) error {
			var encErr error
			ciphertext, encErr = Encrypt(k, plaintext, nil)
			return encErr
		})
		require.NoError(t, err)
		assert.NotEmpty(t, ciphertext)

		// Decrypt with same key to verify correctness.
		err = s.Use(func(k []byte) error {
			got, decErr := Decrypt(k, ciphertext, nil)
			if decErr != nil {
				return decErr
			}
			assert.Equal(t, plaintext, got)
			return nil
		})
		require.NoError(t, err)
	})
}

// TestKeyStore_Zero covers permanent key destruction.
func TestKeyStore_Zero(t *testing.T) {
	t.Run("IsSet returns false after Zero", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))
		s.Zero()
		assert.False(t, s.IsSet())
	})

	t.Run("Use returns ErrKeyNotSet after Zero", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))
		s.Zero()
		err := s.Use(func(_ []byte) error { return nil })
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrKeyNotSet)
	})

	t.Run("double Zero does not panic", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))
		assert.NotPanics(t, func() {
			s.Zero()
			s.Zero()
		})
	})

	t.Run("Zero on unset store does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			NewKeyStore().Zero()
		})
	})

	t.Run("Set after Zero succeeds", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))
		s.Zero()
		require.NoError(t, s.Set(mustGenerateKey(t)))
		assert.True(t, s.IsSet())
	})
}

// TestKeyStore_Concurrent verifies thread safety under the race detector.
func TestKeyStore_Concurrent(t *testing.T) {
	t.Run("concurrent Use calls succeed", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))

		const goroutines = 50
		errs := make([]error, goroutines)
		var wg sync.WaitGroup
		for i := range goroutines {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				errs[i] = s.Use(func(k []byte) error {
					if len(k) != KeySize {
						return errors.New("wrong key length")
					}
					return nil
				})
			}(i)
		}
		wg.Wait()

		for i, err := range errs {
			assert.NoError(t, err, "goroutine %d", i)
		}
	})

	t.Run("concurrent Set and Use do not race", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))

		var wg sync.WaitGroup
		for range 20 {
			wg.Add(2)
			go func() {
				defer wg.Done()
				_ = s.Use(func(_ []byte) error { return nil })
			}()
			go func() {
				defer wg.Done()
				_ = s.Set(mustGenerateKey(t))
			}()
		}
		wg.Wait()
	})

	t.Run("concurrent Zero and Use return consistent state", func(t *testing.T) {
		s := NewKeyStore()
		require.NoError(t, s.Set(mustGenerateKey(t)))

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Zero()
		}()

		// Use may return either nil or ErrKeyNotSet depending on ordering — both are valid.
		err := s.Use(func(_ []byte) error { return nil })
		wg.Wait()
		if err != nil {
			assert.ErrorIs(t, err, ErrKeyNotSet)
		}
	})
}

// mustGenerateKey is a test helper that generates a key and fails the test on error.
func mustGenerateKey(t *testing.T) []byte {
	t.Helper()
	key, err := GenerateKey()
	require.NoError(t, err)
	return key
}

// mustEncrypt is a test helper that encrypts (with no AAD) and fails the test on error.
func mustEncrypt(t *testing.T, key, plaintext []byte) []byte {
	t.Helper()
	ct, err := Encrypt(key, plaintext, nil)
	require.NoError(t, err)
	return ct
}

// cloneKey returns a copy of key that is unaffected by zeroing.
func cloneKey(key []byte) []byte {
	clone := make([]byte, len(key))
	copy(clone, key)
	return clone
}

// isZeroed reports whether all bytes in b are 0x00.
func isZeroed(b []byte) bool {
	return bytes.Equal(b, make([]byte, len(b)))
}
