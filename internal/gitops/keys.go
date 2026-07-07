package gitops

import (
	"fmt"
	"log/slog"

	"filippo.io/age"
	icrypto "github.com/bytepunx/signet/internal/crypto"
)

// GenerateAgeKey generates a new X25519 age keypair.
// The private key is immediately encrypted under the current master key so
// plaintext key material never persists beyond this function's stack frame.
// Returns the bech32 public key string and the encrypted private key ciphertext.
func GenerateAgeKey(keys keyUnwrapper) (pubKey string, encPrivKey []byte, err error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return "", nil, fmt.Errorf("generate age identity: %w", err)
	}

	// Handle the private key as bytes end-to-end (not string) so it can
	// actually be zeroed — Go strings are immutable and cannot be scrubbed.
	privKeyBytes := []byte(id.String()) // AGE-SECRET-KEY-1...
	defer ZeroBytes(privKeyBytes)
	pubKey = id.Recipient().String() // age1...

	// Encrypt the private key using the master key, bound to its own public
	// key so it cannot be swapped for a different key's ciphertext.
	var ciphertext []byte
	if err := keys.Use(func(masterKey []byte) error {
		ct, err := icrypto.Encrypt(masterKey, privKeyBytes, icrypto.BindAAD(icrypto.AADSOPSAgeKey, pubKey))
		if err != nil {
			return err
		}
		ciphertext = ct
		return nil
	}); err != nil {
		return "", nil, fmt.Errorf("encrypt age private key: %w", err)
	}

	return pubKey, ciphertext, nil
}

// DecryptAgeKey decrypts an age private key that was encrypted by GenerateAgeKey.
// Returns an age.Identity ready for use in DecryptFile.
// The caller must not retain the Identity after the operation is complete;
// the underlying key material lives in process memory until GC.
func DecryptAgeKey(keys keyUnwrapper, pubKey string, encPrivKey []byte) (age.Identity, error) {
	aad := icrypto.BindAAD(icrypto.AADSOPSAgeKey, pubKey)
	var id age.Identity
	if err := keys.Use(func(masterKey []byte) error {
		plaintext, legacy, err := icrypto.DecryptWithFallback(masterKey, encPrivKey, aad)
		if err != nil {
			return fmt.Errorf("decrypt age private key: %w", err)
		}
		defer ZeroBytes(plaintext)
		if legacy {
			slog.Warn("sops age private key decrypted via legacy unbound fallback; will be re-bound on next signet sops-key rotate", "public_key", pubKey)
		}
		parsed, err := age.ParseX25519Identity(string(plaintext))
		if err != nil {
			return fmt.Errorf("parse age identity: %w", err)
		}
		id = parsed
		return nil
	}); err != nil {
		return nil, err
	}
	return id, nil
}
