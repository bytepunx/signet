package api

import (
	"context"
	"errors"
	"fmt"

	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
)

// ErrKeyCheckFailed is returned when a candidate master key decrypts
// successfully in isolation but does not match the stored key-check value —
// meaning it is the wrong key for this deployment (e.g. an operator supplied
// a share set or key from a different signet instance).
var ErrKeyCheckFailed = errors.New("key check failed: this key does not match the stored key-check value")

// kcvPlaintext is the fixed constant encrypted to form the key-check value.
// Its content is not sensitive; only successful decryption matters.
const kcvPlaintext = "signet-key-check-value-v1"

// KeyCheckStore is the minimal store dependency for VerifyOrInitKeyCheckValue.
// *store.Store satisfies it directly.
type KeyCheckStore interface {
	GetKeyCheckValue(ctx context.Context) ([]byte, error)
	PutKeyCheckValue(ctx context.Context, ciphertext []byte) error
}

// KeyUser is the minimal key dependency for VerifyOrInitKeyCheckValue.
// *crypto.KeyStore satisfies it directly.
type KeyUser interface {
	Use(fn func([]byte) error) error
}

// VerifyOrInitKeyCheckValue is called synchronously immediately after any
// unseal operation succeeds, before the operation is reported to the caller
// as complete. On a brand new deployment (no key-check value has ever been
// created) it mints one under the newly-loaded key. On every subsequent
// unseal it verifies the loaded key can decrypt the existing key-check value;
// on mismatch it returns ErrKeyCheckFailed and the caller must re-seal.
//
// Exported so both AdminServer (the manual/Shamir unseal RPCs) and the
// signetd Kubernetes auto-unseal path share exactly one key-check
// implementation.
func VerifyOrInitKeyCheckValue(ctx context.Context, st KeyCheckStore, keys KeyUser) error {
	aad := icrypto.BindAAD(icrypto.AADKeyCheckValue)

	existing, err := st.GetKeyCheckValue(ctx)
	if errors.Is(err, store.ErrNotFound) {
		var ct []byte
		if err := keys.Use(func(masterKey []byte) error {
			c, encErr := icrypto.Encrypt(masterKey, []byte(kcvPlaintext), aad)
			ct = c
			return encErr
		}); err != nil {
			return fmt.Errorf("create key-check value: %w", err)
		}
		if err := st.PutKeyCheckValue(ctx, ct); err != nil {
			return fmt.Errorf("persist key-check value: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("load key-check value: %w", err)
	}

	return keys.Use(func(masterKey []byte) error {
		_, decErr := icrypto.Decrypt(masterKey, existing, aad)
		if decErr != nil {
			return ErrKeyCheckFailed
		}
		return nil
	})
}
