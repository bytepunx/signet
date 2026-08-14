//go:build integration

package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cleanRepoSecretTracking(t *testing.T, s *Store) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), "DELETE FROM secrets")
	require.NoError(t, err)
	_, err = s.pool.Exec(context.Background(), "DELETE FROM configs")
	require.NoError(t, err)
	_, err = s.pool.Exec(context.Background(), "DELETE FROM git_repositories")
	require.NoError(t, err)
}

func newTestRepository(t *testing.T, s *Store, name string) *Repository {
	t.Helper()
	r := &Repository{
		Name:                   name,
		RepoURL:                "git@example.com:" + name + ".git",
		Branch:                 "main",
		SecretsPath:            "secrets/",
		EncryptedWebhookSecret: []byte("webhook-secret"),
		EncryptedDeployKey:     []byte("deploy-key"),
	}
	require.NoError(t, s.PutRepository(context.Background(), r))
	return r
}

// TestDeleteSecret_RemovingRepoDoesNotCascade verifies the migration's
// explicit ON DELETE SET NULL choice: removing a repository registration
// must not delete the secrets it synced, only detach their attribution.
func TestDeleteSecret_RemovingRepoDoesNotCascade(t *testing.T) {
	s := newTestStore(t)
	cleanRepoSecretTracking(t, s)
	ctx := context.Background()

	repo := newTestRepository(t, s, "repo-to-remove")
	require.NoError(t, s.PutSecret(ctx, &Secret{
		Namespace: "ns", Service: "svc", Name: "survives",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"), RepoID: repo.ID,
	}))

	_, err := s.pool.Exec(ctx, "DELETE FROM git_repositories WHERE id = $1", repo.ID)
	require.NoError(t, err)

	got, err := s.GetSecret(ctx, "ns", "svc", "survives")
	require.NoError(t, err, "the secret itself must survive its repository being deregistered")
	assert.Equal(t, "ct", string(got.Ciphertext))
}

// TestUpdateSecretRepoID_UpdatesLatestVersionOnly verifies that
// UpdateSecretRepoID (used by storeSecret's dedup path to keep repo_id
// current even when the write itself is skipped as unchanged) updates only
// the latest version's repo_id, without creating a new version or touching
// any other field.
func TestUpdateSecretRepoID_UpdatesLatestVersionOnly(t *testing.T) {
	s := newTestStore(t)
	cleanRepoSecretTracking(t, s)
	ctx := context.Background()

	repoOld := newTestRepository(t, s, "repo-old-update")
	repoNew := newTestRepository(t, s, "repo-new-update")

	sec := &Secret{
		Namespace: "ns", Service: "svc", Name: "stable",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"), RepoID: repoOld.ID,
	}
	require.NoError(t, s.PutSecret(ctx, sec))
	require.Equal(t, 1, sec.Version)

	require.NoError(t, s.UpdateSecretRepoID(ctx, "ns", "svc", "stable", repoNew.ID))

	got, err := s.GetSecret(ctx, "ns", "svc", "stable")
	require.NoError(t, err)
	assert.Equal(t, 1, got.Version, "must not create a new version")
	assert.Equal(t, "ct", string(got.Ciphertext), "must not touch the ciphertext")

	var repoID string
	require.NoError(t, s.pool.QueryRow(ctx,
		"SELECT repo_id FROM secrets WHERE namespace = $1 AND service = $2 AND secret_name = $3 AND version = 1",
		"ns", "svc", "stable",
	).Scan(&repoID))
	assert.Equal(t, repoNew.ID, repoID)
}

// TestUpdateSecretRepoID_NotFound verifies the not-found contract for a
// secret that doesn't exist at all.
func TestUpdateSecretRepoID_NotFound(t *testing.T) {
	s := newTestStore(t)
	cleanRepoSecretTracking(t, s)
	ctx := context.Background()

	repo := newTestRepository(t, s, "repo-notfound-update")
	err := s.UpdateSecretRepoID(ctx, "ns", "svc", "nonexistent", repo.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}
