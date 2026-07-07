//go:build integration

package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cleanKEKs(t *testing.T, s *Store) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), "DELETE FROM secrets")
	require.NoError(t, err)
	_, err = s.pool.Exec(context.Background(), "DELETE FROM key_encryption_keys")
	require.NoError(t, err)
	_, err = s.pool.Exec(context.Background(), "DELETE FROM key_check_value")
	require.NoError(t, err)
}

func TestPutKEK_PopulatesIDAndCreatedAt(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	k := &KEK{WrappedKEK: []byte("wrapped"), IsActive: true}
	require.NoError(t, s.PutKEK(context.Background(), k))
	assert.NotEmpty(t, k.ID)
	assert.False(t, k.CreatedAt.IsZero())
}

func TestPutKEK_EmptyWrappedKEK(t *testing.T) {
	s := newTestStore(t)
	err := s.PutKEK(context.Background(), &KEK{IsActive: true})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetActiveKEK_NotFoundWhenNoneProvisioned(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	_, err := s.GetActiveKEK(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetActiveKEK_ReturnsTheActiveOne(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	inactive := &KEK{WrappedKEK: []byte("old"), IsActive: false}
	require.NoError(t, s.PutKEK(context.Background(), inactive))
	active := &KEK{WrappedKEK: []byte("new"), IsActive: true}
	require.NoError(t, s.PutKEK(context.Background(), active))

	got, err := s.GetActiveKEK(context.Background())
	require.NoError(t, err)
	assert.Equal(t, active.ID, got.ID)
	assert.Equal(t, []byte("new"), got.WrappedKEK)
}

func TestGetKEKByID_FindsInactiveKEKs(t *testing.T) {
	// Rotation must still be able to look up an old, now-inactive KEK to
	// unwrap DEKs that were wrapped before the most recent rotation.
	s := newTestStore(t)
	cleanKEKs(t, s)

	k := &KEK{WrappedKEK: []byte("wrapped"), IsActive: false}
	require.NoError(t, s.PutKEK(context.Background(), k))

	got, err := s.GetKEKByID(context.Background(), k.ID)
	require.NoError(t, err)
	assert.Equal(t, k.ID, got.ID)
	assert.False(t, got.IsActive)
}

func TestGetKEKByID_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetKEKByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListKEKs_OrderedByCreatedAtDescending(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	first := &KEK{WrappedKEK: []byte("first"), IsActive: false}
	require.NoError(t, s.PutKEK(context.Background(), first))
	second := &KEK{WrappedKEK: []byte("second"), IsActive: true}
	require.NoError(t, s.PutKEK(context.Background(), second))

	keks, err := s.ListKEKs(context.Background())
	require.NoError(t, err)
	require.Len(t, keks, 2)
	assert.Equal(t, second.ID, keks[0].ID)
	assert.Equal(t, first.ID, keks[1].ID)
}

func TestDeactivateKEK_SetsInactiveAndTimestamp(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	k := &KEK{WrappedKEK: []byte("wrapped"), IsActive: true}
	require.NoError(t, s.PutKEK(context.Background(), k))

	require.NoError(t, s.DeactivateKEK(context.Background(), k.ID))

	got, err := s.GetKEKByID(context.Background(), k.ID)
	require.NoError(t, err)
	assert.False(t, got.IsActive)
	require.NotNil(t, got.DeactivatedAt)
}

func TestDeactivateKEK_NotFound(t *testing.T) {
	s := newTestStore(t)
	err := s.DeactivateKEK(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCountSecretsUsingKEK(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	k := &KEK{WrappedKEK: []byte("wrapped"), IsActive: true}
	require.NoError(t, s.PutKEK(context.Background(), k))

	n, err := s.CountSecretsUsingKEK(context.Background(), k.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "key",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"), KEKID: k.ID,
	}))

	n, err = s.CountSecretsUsingKEK(context.Background(), k.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestDeleteKEK_OK(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	k := &KEK{WrappedKEK: []byte("wrapped"), IsActive: false}
	require.NoError(t, s.PutKEK(context.Background(), k))

	require.NoError(t, s.DeleteKEK(context.Background(), k.ID))

	_, err := s.GetKEKByID(context.Background(), k.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteKEK_NotFound(t *testing.T) {
	s := newTestStore(t)
	err := s.DeleteKEK(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSecret_RoundtripsKEKID(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	k := &KEK{WrappedKEK: []byte("wrapped"), IsActive: true}
	require.NoError(t, s.PutKEK(context.Background(), k))

	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "key",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"), KEKID: k.ID,
	}))

	got, err := s.GetSecret(context.Background(), "ns", "svc", "key")
	require.NoError(t, err)
	assert.Equal(t, k.ID, got.KEKID)
}

func TestSecret_EmptyKEKIDMeansLegacyDirectMasterWrap(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "key",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"),
	}))

	got, err := s.GetSecret(context.Background(), "ns", "svc", "key")
	require.NoError(t, err)
	assert.Empty(t, got.KEKID)
}

func TestKeyCheckValue_NotFoundBeforeCreation(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	_, err := s.GetKeyCheckValue(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestKeyCheckValue_PutThenGet(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	require.NoError(t, s.PutKeyCheckValue(context.Background(), []byte("kcv-ciphertext")))

	got, err := s.GetKeyCheckValue(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("kcv-ciphertext"), got)
}

func TestKeyCheckValue_PutEmptyRejected(t *testing.T) {
	s := newTestStore(t)
	err := s.PutKeyCheckValue(context.Background(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestReplaceKeyCheckValue_OverwritesExisting(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	require.NoError(t, s.PutKeyCheckValue(context.Background(), []byte("old")))
	require.NoError(t, s.ReplaceKeyCheckValue(context.Background(), []byte("new")))

	got, err := s.GetKeyCheckValue(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("new"), got)
}

func TestReplaceKeyCheckValue_CreatesIfAbsent(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	require.NoError(t, s.ReplaceKeyCheckValue(context.Background(), []byte("first")))

	got, err := s.GetKeyCheckValue(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("first"), got)
}

func TestListSecretKeyRefs_ReturnsEveryRow(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	k := &KEK{WrappedKEK: []byte("wrapped"), IsActive: true}
	require.NoError(t, s.PutKEK(context.Background(), k))

	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "a",
		EncryptedDEK: []byte("dek-a"), Ciphertext: []byte("ct-a"), KEKID: k.ID,
	}))
	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "b",
		EncryptedDEK: []byte("dek-b"), Ciphertext: []byte("ct-b"),
	}))

	refs, err := s.ListSecretKeyRefs(context.Background())
	require.NoError(t, err)
	require.Len(t, refs, 2)

	byName := map[string]SecretKeyRef{}
	for _, r := range refs {
		byName[r.Name] = r
	}
	assert.Equal(t, k.ID, byName["a"].KEKID)
	assert.Empty(t, byName["b"].KEKID)
}

func TestUpdateSecretDEK_RewritesDEKAndKEKReference(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	oldKEK := &KEK{WrappedKEK: []byte("old-wrapped"), IsActive: false}
	require.NoError(t, s.PutKEK(context.Background(), oldKEK))
	newKEK := &KEK{WrappedKEK: []byte("new-wrapped"), IsActive: true}
	require.NoError(t, s.PutKEK(context.Background(), newKEK))

	sec := &Secret{
		Namespace: "ns", Service: "svc", Name: "key",
		EncryptedDEK: []byte("dek-under-old"), Ciphertext: []byte("ct"), KEKID: oldKEK.ID,
	}
	require.NoError(t, s.PutSecret(context.Background(), sec))

	require.NoError(t, s.UpdateSecretDEK(context.Background(), "ns", "svc", "key", sec.Version, []byte("dek-under-new"), newKEK.ID))

	got, err := s.GetSecret(context.Background(), "ns", "svc", "key")
	require.NoError(t, err)
	assert.Equal(t, []byte("dek-under-new"), got.EncryptedDEK)
	assert.Equal(t, newKEK.ID, got.KEKID)
	// Ciphertext and version must be untouched by a DEK rewrap.
	assert.Equal(t, []byte("ct"), got.Ciphertext)
	assert.Equal(t, sec.Version, got.Version)
}

func TestUpdateSecretDEK_NotFound(t *testing.T) {
	s := newTestStore(t)
	err := s.UpdateSecretDEK(context.Background(), "ns", "svc", "missing", 1, []byte("dek"), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRewrapKEKsAndKCV_UpdatesAllAtomically(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	k1 := &KEK{WrappedKEK: []byte("k1-old"), IsActive: false}
	require.NoError(t, s.PutKEK(context.Background(), k1))
	k2 := &KEK{WrappedKEK: []byte("k2-old"), IsActive: true}
	require.NoError(t, s.PutKEK(context.Background(), k2))
	require.NoError(t, s.PutKeyCheckValue(context.Background(), []byte("kcv-old")))

	err := s.RewrapKEKsAndKCV(context.Background(), []KEKRewrap{
		{ID: k1.ID, WrappedKEK: []byte("k1-new")},
		{ID: k2.ID, WrappedKEK: []byte("k2-new")},
	}, []byte("kcv-new"))
	require.NoError(t, err)

	got1, err := s.GetKEKByID(context.Background(), k1.ID)
	require.NoError(t, err)
	assert.Equal(t, []byte("k1-new"), got1.WrappedKEK)

	got2, err := s.GetKEKByID(context.Background(), k2.ID)
	require.NoError(t, err)
	assert.Equal(t, []byte("k2-new"), got2.WrappedKEK)

	kcv, err := s.GetKeyCheckValue(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("kcv-new"), kcv)
}

func TestRewrapKEKsAndKCV_RollsBackOnUnknownKEK(t *testing.T) {
	s := newTestStore(t)
	cleanKEKs(t, s)

	k1 := &KEK{WrappedKEK: []byte("k1-old"), IsActive: true}
	require.NoError(t, s.PutKEK(context.Background(), k1))
	require.NoError(t, s.PutKeyCheckValue(context.Background(), []byte("kcv-old")))

	err := s.RewrapKEKsAndKCV(context.Background(), []KEKRewrap{
		{ID: k1.ID, WrappedKEK: []byte("k1-new")},
		{ID: "00000000-0000-0000-0000-000000000000", WrappedKEK: []byte("nope")},
	}, []byte("kcv-new"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)

	// Neither the valid KEK nor the KCV should have been updated: the whole
	// rotation is atomic.
	got1, err := s.GetKEKByID(context.Background(), k1.ID)
	require.NoError(t, err)
	assert.Equal(t, []byte("k1-old"), got1.WrappedKEK)

	kcv, err := s.GetKeyCheckValue(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("kcv-old"), kcv)
}

func TestPutRepository_DuplicateNameRejected(t *testing.T) {
	s := newTestStore(t)
	_, err := s.pool.Exec(context.Background(), "DELETE FROM git_repositories")
	require.NoError(t, err)

	r1 := &Repository{
		Name: "infra-secrets", RepoURL: "git@example.com:org/repo",
		EncryptedWebhookSecret: []byte("wh"), EncryptedDeployKey: []byte("dk"),
	}
	require.NoError(t, s.PutRepository(context.Background(), r1))

	r2 := &Repository{
		Name: "infra-secrets", RepoURL: "git@example.com:org/other",
		EncryptedWebhookSecret: []byte("wh2"), EncryptedDeployKey: []byte("dk2"),
	}
	err = s.PutRepository(context.Background(), r2)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAlreadyExists)
}
