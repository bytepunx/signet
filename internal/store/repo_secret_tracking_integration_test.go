//go:build integration

package store

import (
	"context"
	"encoding/json"
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

// TestListSecretKeysForRepo_ScopesToRepo verifies that only secrets whose
// latest version is attributed to the given repo are returned — this is
// what FullSync's deletion detection relies on to avoid ever touching a
// secret that belongs to a different registered repository.
func TestListSecretKeysForRepo_ScopesToRepo(t *testing.T) {
	s := newTestStore(t)
	cleanRepoSecretTracking(t, s)
	ctx := context.Background()

	repoA := newTestRepository(t, s, "repo-a-scope")
	repoB := newTestRepository(t, s, "repo-b-scope")

	require.NoError(t, s.PutSecret(ctx, &Secret{
		Namespace: "ns", Service: "svc", Name: "from-a",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"), RepoID: repoA.ID,
	}))
	require.NoError(t, s.PutSecret(ctx, &Secret{
		Namespace: "ns", Service: "svc", Name: "from-b",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"), RepoID: repoB.ID,
	}))
	require.NoError(t, s.PutSecret(ctx, &Secret{
		Namespace: "ns", Service: "svc", Name: "unattributed",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"),
	}))

	keysA, err := s.ListSecretKeysForRepo(ctx, repoA.ID)
	require.NoError(t, err)
	assert.Equal(t, []SecretKey{{Namespace: "ns", Service: "svc", Name: "from-a"}}, keysA)

	keysB, err := s.ListSecretKeysForRepo(ctx, repoB.ID)
	require.NoError(t, err)
	assert.Equal(t, []SecretKey{{Namespace: "ns", Service: "svc", Name: "from-b"}}, keysB)
}

// TestListSecretKeysForRepo_UsesLatestVersionOnly verifies that a secret
// re-synced by a different repo (a new version with a different repo_id) is
// attributed to whichever repo owns its LATEST version, not any earlier
// one — otherwise the older repo's next sync would wrongly conclude the
// secret still belongs to it and never let it go, while the new repo would
// never see it as its own to manage either.
func TestListSecretKeysForRepo_UsesLatestVersionOnly(t *testing.T) {
	s := newTestStore(t)
	cleanRepoSecretTracking(t, s)
	ctx := context.Background()

	repoA := newTestRepository(t, s, "repo-a-versions")
	repoB := newTestRepository(t, s, "repo-b-versions")

	require.NoError(t, s.PutSecret(ctx, &Secret{
		Namespace: "ns", Service: "svc", Name: "migrated",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct-v1"), RepoID: repoA.ID,
	}))
	// Re-synced under repo B — a new version, same (namespace, service, name).
	require.NoError(t, s.PutSecret(ctx, &Secret{
		Namespace: "ns", Service: "svc", Name: "migrated",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct-v2"), RepoID: repoB.ID,
	}))

	keysA, err := s.ListSecretKeysForRepo(ctx, repoA.ID)
	require.NoError(t, err)
	assert.Empty(t, keysA, "repo A no longer owns the latest version of this secret")

	keysB, err := s.ListSecretKeysForRepo(ctx, repoB.ID)
	require.NoError(t, err)
	assert.Equal(t, []SecretKey{{Namespace: "ns", Service: "svc", Name: "migrated"}}, keysB)
}

// TestListConfigKeysForRepo_ScopesToRepo mirrors the secret test above for
// the configs table.
func TestListConfigKeysForRepo_ScopesToRepo(t *testing.T) {
	s := newTestStore(t)
	cleanRepoSecretTracking(t, s)
	ctx := context.Background()

	repoA := newTestRepository(t, s, "repo-a-config")
	repoB := newTestRepository(t, s, "repo-b-config")

	require.NoError(t, s.PutServiceConfig(ctx, "ns-a", "svc", json.RawMessage(`{"k":"a"}`), repoA.ID))
	require.NoError(t, s.PutServiceConfig(ctx, "ns-b", "svc", json.RawMessage(`{"k":"b"}`), repoB.ID))
	require.NoError(t, s.PutServiceConfig(ctx, "ns-none", "svc", json.RawMessage(`{"k":"c"}`), ""))

	keysA, err := s.ListConfigKeysForRepo(ctx, repoA.ID)
	require.NoError(t, err)
	assert.Equal(t, []ConfigKey{{Namespace: "ns-a", Service: "svc"}}, keysA)

	keysB, err := s.ListConfigKeysForRepo(ctx, repoB.ID)
	require.NoError(t, err)
	assert.Equal(t, []ConfigKey{{Namespace: "ns-b", Service: "svc"}}, keysB)
}

// TestListConfigKeysForRepo_ReattributionOnReSync verifies that re-syncing
// an existing config under a different repo updates its attribution (via
// PutServiceConfig's ON CONFLICT ... SET repo_id = excluded.repo_id), the
// same latest-wins guarantee TestListSecretKeysForRepo_UsesLatestVersionOnly
// checks for secrets.
func TestListConfigKeysForRepo_ReattributionOnReSync(t *testing.T) {
	s := newTestStore(t)
	cleanRepoSecretTracking(t, s)
	ctx := context.Background()

	repoA := newTestRepository(t, s, "repo-a-config-resync")
	repoB := newTestRepository(t, s, "repo-b-config-resync")

	require.NoError(t, s.PutServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"v":1}`), repoA.ID))
	require.NoError(t, s.PutServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"v":2}`), repoB.ID))

	keysA, err := s.ListConfigKeysForRepo(ctx, repoA.ID)
	require.NoError(t, err)
	assert.Empty(t, keysA)

	keysB, err := s.ListConfigKeysForRepo(ctx, repoB.ID)
	require.NoError(t, err)
	assert.Equal(t, []ConfigKey{{Namespace: "ns", Service: "svc"}}, keysB)
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

	keysOld, err := s.ListSecretKeysForRepo(ctx, repoOld.ID)
	require.NoError(t, err)
	assert.Empty(t, keysOld)

	keysNew, err := s.ListSecretKeysForRepo(ctx, repoNew.ID)
	require.NoError(t, err)
	assert.Equal(t, []SecretKey{{Namespace: "ns", Service: "svc", Name: "stable"}}, keysNew)
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
