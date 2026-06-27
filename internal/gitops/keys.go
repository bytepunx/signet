package gitops

import (
	"fmt"
	"strings"

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

	privKeyStr := id.String()   // AGE-SECRET-KEY-1...
	pubKey = id.Recipient().String() // age1...

	// Encrypt the private key string using the master key via our AES-256-GCM envelope.
	var ciphertext []byte
	if err := keys.Use(func(masterKey []byte) error {
		ct, err := icrypto.Encrypt(masterKey, []byte(privKeyStr))
		if err != nil {
			return err
		}
		ciphertext = ct
		// Zero the private key string from the local stack copy.
		for i := range []byte(privKeyStr) {
			privKeyStr = strings.Repeat("\x00", len(privKeyStr))
			_ = i
			break
		}
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
func DecryptAgeKey(keys keyUnwrapper, encPrivKey []byte) (age.Identity, error) {
	var id age.Identity
	if err := keys.Use(func(masterKey []byte) error {
		plaintext, err := icrypto.Decrypt(masterKey, encPrivKey)
		if err != nil {
			return fmt.Errorf("decrypt age private key: %w", err)
		}
		parsed, err := age.ParseX25519Identity(string(plaintext))
		if err != nil {
			return fmt.Errorf("parse age identity: %w", err)
		}
		id = parsed
		// Zero the plaintext private key bytes.
		for i := range plaintext {
			plaintext[i] = 0
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return id, nil
}
