package gitops

import (
	"context"
	"testing"

	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// storeUnderActiveKEK persists a secret in st, encrypted exactly the way
// storeSecret would, under the given active KEK. Returns the AAD used.
func storeUnderActiveKEK(t *testing.T, st *statefulKEKStore, kekID string, kekBytes []byte, namespace, service, name string, plaintext []byte) []byte {
	t.Helper()
	aad := icrypto.BindAAD(icrypto.AADSecret, namespace, service, name)
	dek, err := icrypto.GenerateKey()
	require.NoError(t, err)
	ct, err := icrypto.Encrypt(dek, plaintext, aad)
	require.NoError(t, err)
	encDEK, err := icrypto.WrapKey(kekBytes, dek, aad)
	require.NoError(t, err)
	require.NoError(t, st.PutSecret(context.Background(), &store.Secret{
		Namespace: namespace, Service: service, Name: name,
		EncryptedDEK: encDEK, Ciphertext: ct, KEKID: kekID,
	}))
	return aad
}

// TestIsUnchanged_TrueWhenPlaintextAndKEKMatch is the M-4 core case: identical
// plaintext already on the current KEK epoch must be reported unchanged.
func TestIsUnchanged_TrueWhenPlaintextAndKEKMatch(t *testing.T) {
	st := &statefulKEKStore{}
	keys := &mockKeys{}
	s := NewSyncer(st, keys, nil, "")

	kekID, kekBytes, err := activeKEK(context.Background(), st, keys)
	require.NoError(t, err)
	defer ZeroBytes(kekBytes)

	plaintext := []byte("same-value")
	aad := storeUnderActiveKEK(t, st, kekID, kekBytes, "ns", "svc", "key", plaintext)

	assert.True(t, s.isUnchanged(context.Background(), "ns", "svc", "key", plaintext, aad, kekID, kekBytes))
	assert.Equal(t, 1, st.putSecrets, "only the setup call should have written")
}

func TestIsUnchanged_FalseWhenPlaintextDiffers(t *testing.T) {
	st := &statefulKEKStore{}
	keys := &mockKeys{}
	s := NewSyncer(st, keys, nil, "")

	kekID, kekBytes, err := activeKEK(context.Background(), st, keys)
	require.NoError(t, err)
	defer ZeroBytes(kekBytes)

	aad := storeUnderActiveKEK(t, st, kekID, kekBytes, "ns", "svc", "key", []byte("old-value"))

	assert.False(t, s.isUnchanged(context.Background(), "ns", "svc", "key", []byte("new-value"), aad, kekID, kekBytes))
}

func TestIsUnchanged_FalseWhenSecretNotFound(t *testing.T) {
	st := &statefulKEKStore{}
	s := NewSyncer(st, &mockKeys{}, nil, "")

	assert.False(t, s.isUnchanged(context.Background(), "ns", "svc", "missing", []byte("x"), nil, "kek-1", make([]byte, icrypto.KeySize)))
}

// TestIsUnchanged_FalseWhenKEKIDMismatch_ForcesMigrationRewrite is the
// crucial M-4/H-1/M-1 interaction test: a secret still on an old (rotated
// away) KEK must never be reported unchanged, even with matching plaintext,
// so the AAD/KEK migration can converge instead of being permanently
// suppressed by the dedup optimization.
func TestIsUnchanged_FalseWhenKEKIDMismatch_ForcesMigrationRewrite(t *testing.T) {
	st := &statefulKEKStore{}
	oldKEKBytes, err := icrypto.GenerateKey()
	require.NoError(t, err)
	_ = storeUnderActiveKEK(t, st, "old-kek-id", oldKEKBytes, "ns", "svc", "key", []byte("same-value"))

	s := NewSyncer(st, &mockKeys{}, nil, "")
	aad := icrypto.BindAAD(icrypto.AADSecret, "ns", "svc", "key")
	currentKEKBytes := make([]byte, icrypto.KeySize)

	assert.False(t, s.isUnchanged(context.Background(), "ns", "svc", "key", []byte("same-value"), aad, "kek-1-current", currentKEKBytes),
		"a secret on a non-active KEK must always be treated as changed")
}

// TestIsUnchanged_FalseWhenLegacyEmptyKEKID is the H-1 migration case: a
// secret predating the KEK tier entirely (direct master-key wrap) must never
// be reported unchanged, so it gets migrated onto the KEK/AAD scheme on its
// next sync.
func TestIsUnchanged_FalseWhenLegacyEmptyKEKID(t *testing.T) {
	st := &statefulKEKStore{}
	require.NoError(t, st.PutSecret(context.Background(), &store.Secret{
		Namespace: "ns", Service: "svc", Name: "key",
		EncryptedDEK: []byte("whatever"), Ciphertext: []byte("whatever"), KEKID: "",
	}))
	s := NewSyncer(st, &mockKeys{}, nil, "")

	assert.False(t, s.isUnchanged(context.Background(), "ns", "svc", "key", []byte("same-value"), nil, "kek-1", make([]byte, icrypto.KeySize)))
}
