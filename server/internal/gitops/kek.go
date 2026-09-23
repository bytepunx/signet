package gitops

import (
	"context"
	"errors"
	"fmt"

	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
)

// activeKEK returns the id and plaintext bytes of the current active
// key-encryption-key, generating and persisting one under the master key if
// none exists yet (first use on a fresh deployment). The caller must zero the
// returned key bytes when done.
func activeKEK(ctx context.Context, st secretStore, keys keyUnwrapper) (id string, kek []byte, err error) {
	rec, err := st.GetActiveKEK(ctx)
	if errors.Is(err, store.ErrNotFound) {
		return bootstrapKEK(ctx, st, keys)
	}
	if err != nil {
		return "", nil, fmt.Errorf("get active kek: %w", err)
	}

	if err := keys.Use(func(masterKey []byte) error {
		plain, uErr := icrypto.UnwrapKey(masterKey, rec.WrappedKEK, icrypto.BindAAD(icrypto.AADKEK))
		if uErr != nil {
			return uErr
		}
		kek = plain
		return nil
	}); err != nil {
		return "", nil, fmt.Errorf("unwrap active kek: %w", err)
	}

	return rec.ID, kek, nil
}

// bootstrapKEK generates a fresh KEK, wraps it under the master key, and
// persists it as the (first and only) active KEK. Called lazily the first
// time a KEK is needed so no explicit operator setup step is required.
func bootstrapKEK(ctx context.Context, st secretStore, keys keyUnwrapper) (id string, kek []byte, err error) {
	newKEK, err := icrypto.GenerateKey()
	if err != nil {
		return "", nil, fmt.Errorf("generate kek: %w", err)
	}

	var wrapped []byte
	if err := keys.Use(func(masterKey []byte) error {
		w, wErr := icrypto.WrapKey(masterKey, cloneBytes(newKEK), icrypto.BindAAD(icrypto.AADKEK))
		wrapped = w
		return wErr
	}); err != nil {
		ZeroBytes(newKEK)
		return "", nil, fmt.Errorf("wrap new kek: %w", err)
	}

	rec := &store.KEK{WrappedKEK: wrapped, IsActive: true}
	if err := st.PutKEK(ctx, rec); err != nil {
		ZeroBytes(newKEK)
		return "", nil, fmt.Errorf("persist new kek: %w", err)
	}

	return rec.ID, newKEK, nil
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
