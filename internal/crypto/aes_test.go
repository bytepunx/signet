package crypto

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncrypt covers output shape, nonce randomness, and key validation.
func TestEncrypt(t *testing.T) {
	key := mustGenerateKey(t)

	t.Run("happy path produces output", func(t *testing.T) {
		ct, err := Encrypt(key, []byte("secret"))
		require.NoError(t, err)
		assert.NotEmpty(t, ct)
	})

	t.Run("output length is nonceSize + len(plaintext) + GCM tag", func(t *testing.T) {
		plaintext := []byte("hello")
		ct, err := Encrypt(key, plaintext)
		require.NoError(t, err)
		// nonce(12) + plaintext(5) + tag(16) = 33
		assert.Len(t, ct, nonceSize+len(plaintext)+16)
	})

	t.Run("empty plaintext is valid", func(t *testing.T) {
		ct, err := Encrypt(key, []byte{})
		require.NoError(t, err)
		// nonce(12) + plaintext(0) + tag(16) = 28
		assert.Len(t, ct, nonceSize+16)
	})

	t.Run("nil plaintext is treated as empty", func(t *testing.T) {
		ct, err := Encrypt(key, nil)
		require.NoError(t, err)
		assert.Len(t, ct, nonceSize+16)
	})

	t.Run("large plaintext (1 MiB)", func(t *testing.T) {
		plaintext := bytes.Repeat([]byte("a"), 1<<20)
		ct, err := Encrypt(key, plaintext)
		require.NoError(t, err)
		assert.Len(t, ct, nonceSize+len(plaintext)+16)
	})

	t.Run("two calls produce different ciphertext", func(t *testing.T) {
		pt := []byte("same plaintext")
		ct1, err := Encrypt(key, pt)
		require.NoError(t, err)
		ct2, err := Encrypt(key, pt)
		require.NoError(t, err)
		assert.NotEqual(t, ct1, ct2, "nonces must differ per call")
	})

	t.Run("nil key returns ErrInvalidKeySize", func(t *testing.T) {
		_, err := Encrypt(nil, []byte("x"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("short key returns ErrInvalidKeySize", func(t *testing.T) {
		_, err := Encrypt(make([]byte, KeySize-1), []byte("x"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("long key returns ErrInvalidKeySize", func(t *testing.T) {
		_, err := Encrypt(make([]byte, KeySize+1), []byte("x"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("empty key returns ErrInvalidKeySize", func(t *testing.T) {
		_, err := Encrypt([]byte{}, []byte("x"))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})
}

// TestDecrypt covers successful decryption and all failure modes.
func TestDecrypt(t *testing.T) {
	key := mustGenerateKey(t)

	t.Run("happy path roundtrip", func(t *testing.T) {
		pt := []byte("the secret value")
		ct := mustEncrypt(t, key, pt)
		got, err := Decrypt(key, ct)
		require.NoError(t, err)
		assert.Equal(t, pt, got)
	})

	t.Run("empty plaintext roundtrip", func(t *testing.T) {
		ct := mustEncrypt(t, key, []byte{})
		got, err := Decrypt(key, ct)
		require.NoError(t, err)
		// gcm.Open returns nil rather than []byte{} for empty plaintext; both mean empty.
		assert.Empty(t, got)
	})

	t.Run("large plaintext roundtrip (1 MiB)", func(t *testing.T) {
		pt := bytes.Repeat([]byte("z"), 1<<20)
		ct := mustEncrypt(t, key, pt)
		got, err := Decrypt(key, ct)
		require.NoError(t, err)
		assert.Equal(t, pt, got)
	})

	t.Run("nil key returns ErrInvalidKeySize", func(t *testing.T) {
		ct := mustEncrypt(t, key, []byte("x"))
		_, err := Decrypt(nil, ct)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("short key returns ErrInvalidKeySize", func(t *testing.T) {
		ct := mustEncrypt(t, key, []byte("x"))
		_, err := Decrypt(make([]byte, KeySize-1), ct)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("long key returns ErrInvalidKeySize", func(t *testing.T) {
		ct := mustEncrypt(t, key, []byte("x"))
		_, err := Decrypt(make([]byte, KeySize+1), ct)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("nil ciphertext returns ErrInvalidCiphertext", func(t *testing.T) {
		_, err := Decrypt(key, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCiphertext)
	})

	t.Run("empty ciphertext returns ErrInvalidCiphertext", func(t *testing.T) {
		_, err := Decrypt(key, []byte{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCiphertext)
	})

	t.Run("ciphertext shorter than nonce returns ErrInvalidCiphertext", func(t *testing.T) {
		_, err := Decrypt(key, make([]byte, nonceSize-1))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCiphertext)
	})

	t.Run("ciphertext with nonce but no tag returns ErrInvalidCiphertext", func(t *testing.T) {
		// nonceSize bytes is not enough to hold a nonce + GCM tag (12+16=28 minimum)
		_, err := Decrypt(key, make([]byte, nonceSize))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCiphertext)
	})

	t.Run("ciphertext one byte short of minimum returns ErrInvalidCiphertext", func(t *testing.T) {
		_, err := Decrypt(key, make([]byte, nonceSize+16-1))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCiphertext)
	})

	t.Run("tampered ciphertext body returns ErrAuthenticationFailed", func(t *testing.T) {
		ct := mustEncrypt(t, key, []byte("secret"))
		ct[len(ct)-1] ^= 0xff
		_, err := Decrypt(key, ct)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed)
	})

	t.Run("tampered nonce returns ErrAuthenticationFailed", func(t *testing.T) {
		ct := mustEncrypt(t, key, []byte("secret"))
		ct[0] ^= 0xff
		_, err := Decrypt(key, ct)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed)
	})

	t.Run("tampered middle byte returns ErrAuthenticationFailed", func(t *testing.T) {
		ct := mustEncrypt(t, key, []byte("secret value"))
		ct[len(ct)/2] ^= 0x01
		_, err := Decrypt(key, ct)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed)
	})

	t.Run("wrong key returns ErrAuthenticationFailed", func(t *testing.T) {
		ct := mustEncrypt(t, key, []byte("secret"))
		wrongKey := mustGenerateKey(t)
		_, err := Decrypt(wrongKey, ct)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed)
	})

	t.Run("all-zero key decrypts ciphertext encrypted with all-zero key", func(t *testing.T) {
		zeroKey := make([]byte, KeySize)
		ct := mustEncrypt(t, zeroKey, []byte("test"))
		got, err := Decrypt(zeroKey, ct)
		require.NoError(t, err)
		assert.Equal(t, []byte("test"), got)
	})
}

// TestEncryptDecryptRoundtrip verifies the full encrypt→decrypt contract.
func TestEncryptDecryptRoundtrip(t *testing.T) {
	plaintexts := []struct {
		name string
		data []byte
	}{
		{"ascii string", []byte("hello, signet")},
		{"binary data", []byte{0x00, 0xff, 0xde, 0xad, 0xbe, 0xef}},
		{"single byte", []byte{0x42}},
		{"unicode", []byte("日本語テスト")},
		{"empty", []byte{}},
	}

	for _, tc := range plaintexts {
		t.Run(tc.name, func(t *testing.T) {
			key := mustGenerateKey(t)
			ct, err := Encrypt(key, tc.data)
			require.NoError(t, err)

			got, err := Decrypt(key, ct)
			require.NoError(t, err)
			if len(tc.data) == 0 {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tc.data, got)
			}
		})
	}

	t.Run("different keys produce different ciphertext for same plaintext", func(t *testing.T) {
		pt := []byte("same plaintext")
		k1, k2 := mustGenerateKey(t), mustGenerateKey(t)
		ct1 := mustEncrypt(t, k1, pt)
		ct2 := mustEncrypt(t, k2, pt)
		assert.NotEqual(t, ct1, ct2)
	})

	t.Run("ciphertext encrypted with key A cannot be decrypted with key B", func(t *testing.T) {
		pt := []byte("secret")
		k1, k2 := mustGenerateKey(t), mustGenerateKey(t)
		ct := mustEncrypt(t, k1, pt)
		_, err := Decrypt(k2, ct)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed)
	})
}

// TestWrapKey covers the key wrapping operation.
func TestWrapKey(t *testing.T) {
	kek := mustGenerateKey(t)
	dek := mustGenerateKey(t)

	t.Run("happy path produces wrapped output", func(t *testing.T) {
		wrapped, err := WrapKey(kek, dek)
		require.NoError(t, err)
		assert.NotEmpty(t, wrapped)
	})

	t.Run("wrapped output differs from the original DEK", func(t *testing.T) {
		wrapped, err := WrapKey(kek, dek)
		require.NoError(t, err)
		assert.NotEqual(t, dek, wrapped)
	})

	t.Run("two wraps of the same DEK produce different output", func(t *testing.T) {
		w1, err := WrapKey(kek, dek)
		require.NoError(t, err)
		w2, err := WrapKey(kek, dek)
		require.NoError(t, err)
		assert.NotEqual(t, w1, w2, "each wrap must use a fresh nonce")
	})

	t.Run("nil kek returns ErrInvalidKeySize", func(t *testing.T) {
		_, err := WrapKey(nil, dek)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("short kek returns ErrInvalidKeySize", func(t *testing.T) {
		_, err := WrapKey(make([]byte, KeySize-1), dek)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("long kek returns ErrInvalidKeySize", func(t *testing.T) {
		_, err := WrapKey(make([]byte, KeySize+1), dek)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("nil dek returns ErrInvalidKeySize", func(t *testing.T) {
		_, err := WrapKey(kek, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("short dek returns ErrInvalidKeySize", func(t *testing.T) {
		_, err := WrapKey(kek, make([]byte, KeySize-1))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("long dek returns ErrInvalidKeySize", func(t *testing.T) {
		_, err := WrapKey(kek, make([]byte, KeySize+1))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})
}

// TestUnwrapKey covers DEK recovery and all failure modes.
func TestUnwrapKey(t *testing.T) {
	kek := mustGenerateKey(t)
	dek := mustGenerateKey(t)

	t.Run("happy path roundtrip with WrapKey", func(t *testing.T) {
		dekCopy := cloneKey(dek)
		wrapped, err := WrapKey(kek, dekCopy)
		require.NoError(t, err)

		got, err := UnwrapKey(kek, wrapped)
		require.NoError(t, err)
		assert.Equal(t, dek, got)
	})

	t.Run("nil kek returns ErrInvalidKeySize", func(t *testing.T) {
		wrapped, _ := WrapKey(kek, cloneKey(dek))
		_, err := UnwrapKey(nil, wrapped)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("short kek returns ErrInvalidKeySize", func(t *testing.T) {
		wrapped, _ := WrapKey(kek, cloneKey(dek))
		_, err := UnwrapKey(make([]byte, KeySize-1), wrapped)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("long kek returns ErrInvalidKeySize", func(t *testing.T) {
		wrapped, _ := WrapKey(kek, cloneKey(dek))
		_, err := UnwrapKey(make([]byte, KeySize+1), wrapped)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("nil wrapped returns ErrInvalidCiphertext", func(t *testing.T) {
		_, err := UnwrapKey(kek, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCiphertext)
	})

	t.Run("empty wrapped returns ErrInvalidCiphertext", func(t *testing.T) {
		_, err := UnwrapKey(kek, []byte{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCiphertext)
	})

	t.Run("truncated wrapped returns ErrInvalidCiphertext", func(t *testing.T) {
		_, err := UnwrapKey(kek, make([]byte, nonceSize-1))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCiphertext)
	})

	t.Run("tampered wrapped returns ErrAuthenticationFailed", func(t *testing.T) {
		wrapped, err := WrapKey(kek, cloneKey(dek))
		require.NoError(t, err)
		wrapped[len(wrapped)-1] ^= 0xff
		_, err = UnwrapKey(kek, wrapped)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed)
	})

	t.Run("wrong kek returns ErrAuthenticationFailed", func(t *testing.T) {
		wrapped, err := WrapKey(kek, cloneKey(dek))
		require.NoError(t, err)
		wrongKEK := mustGenerateKey(t)
		_, err = UnwrapKey(wrongKEK, wrapped)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed)
	})
}

// TestWrapUnwrapRoundtrip verifies the full wrap→unwrap contract across key pairs.
func TestWrapUnwrapRoundtrip(t *testing.T) {
	t.Run("DEK survives roundtrip unchanged", func(t *testing.T) {
		kek := mustGenerateKey(t)
		dek := mustGenerateKey(t)
		original := cloneKey(dek)

		wrapped, err := WrapKey(kek, dek)
		require.NoError(t, err)

		recovered, err := UnwrapKey(kek, wrapped)
		require.NoError(t, err)
		assert.Equal(t, original, recovered)
	})

	t.Run("different KEKs cannot unwrap each other's outputs", func(t *testing.T) {
		kek1 := mustGenerateKey(t)
		kek2 := mustGenerateKey(t)
		dek := mustGenerateKey(t)

		wrapped, err := WrapKey(kek1, dek)
		require.NoError(t, err)

		_, err = UnwrapKey(kek2, wrapped)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrAuthenticationFailed)
	})

	t.Run("re-wrapping the same DEK with same KEK yields different ciphertext", func(t *testing.T) {
		kek := mustGenerateKey(t)
		dek := mustGenerateKey(t)

		w1, err := WrapKey(kek, cloneKey(dek))
		require.NoError(t, err)
		w2, err := WrapKey(kek, cloneKey(dek))
		require.NoError(t, err)

		assert.NotEqual(t, w1, w2, "each wrap must use a fresh nonce")

		// Both must still unwrap to the same DEK.
		r1, err := UnwrapKey(kek, w1)
		require.NoError(t, err)
		r2, err := UnwrapKey(kek, w2)
		require.NoError(t, err)
		assert.Equal(t, r1, r2)
	})
}
