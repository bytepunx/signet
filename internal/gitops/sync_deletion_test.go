package gitops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bytepunx/signet/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSyncableStore builds a statefulKEKStore with one real, registered SOPS
// age key (via GenerateAgeKey, the same helper production code uses), and
// returns its public key alongside the store so callers can pass it to
// sopsEncrypt as the encryption recipient. loadIdentities (called by every
// SyncFromDir/SyncFromPush) needs at least one usable key or every sync
// fails outright with "no age keys configured".
func newSyncableStore(t *testing.T, keys keyUnwrapper) (*statefulKEKStore, string) {
	t.Helper()
	pubKey, encPriv, err := GenerateAgeKey(keys)
	require.NoError(t, err)
	return &statefulKEKStore{
		sopsKeys: []store.SOPSKey{{PublicKey: pubKey, EncryptedPrivateKey: encPriv, IsActive: true}},
	}, pubKey
}

// sopsEncrypt encrypts plaintext with the real sops binary under
// recipientPubKey (an age1... public key already registered with the store
// under test, so DecryptFile can round-trip it via the real decrypt path).
// Skips the calling test if sops isn't on PATH — matches the precedent set
// in sops_test.go's TestDecryptFile_RealSopsCiphertext.
func sopsEncrypt(t *testing.T, dir, relPath, plaintext, recipientPubKey string) {
	t.Helper()
	sopsPath, err := exec.LookPath("sops")
	if err != nil {
		t.Skip("sops binary not found on PATH; skipping real ciphertext test")
	}

	configPath := filepath.Join(dir, ".sops.yaml")
	config := fmt.Sprintf("creation_rules:\n    - path_regex: ^secrets/\n      age: %s\n", recipientPubKey)
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o644))

	fullPath := filepath.Join(dir, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte(plaintext), 0o600))

	cmd := exec.Command(sopsPath, "--config", configPath, "--encrypt", "--in-place", relPath)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "sops encrypt: %s", out)
}

// TestSyncFromDir_DeletesSecretRemovedFromRepo is the regression test for
// the bug this file's Syncer.SyncFromDir repoID/deletion-tracking machinery
// fixes: "signet repo sync" (a full re-walk of the repo, not an incremental
// diff against a webhook payload) used to only ever add/update secrets —
// removing a secret's file from the repo had no effect at all on what
// signet served, forever, until someone deleted it by hand. Confirmed live
// against a real signetd + Postgres instance before this fix existed.
func TestSyncFromDir_DeletesSecretRemovedFromRepo(t *testing.T) {
	dir := t.TempDir()
	keys := &mockKeys{}
	st, pubKey := newSyncableStore(t, keys)
	sopsEncrypt(t, dir, "secrets/ns/svc/keep.yaml", "value: keep-me\n", pubKey)
	sopsEncrypt(t, dir, "secrets/ns/svc/remove.yaml", "value: remove-me\n", pubKey)

	syncer := NewSyncer(st, keys, nil, "")
	ctx := context.Background()

	// First sync: both secrets present, both attributed to "repo-1".
	result, err := syncer.SyncFromDir(ctx, dir, "secrets/", "sha1", "repo-1")
	require.NoError(t, err)
	assert.Equal(t, 2, result.Added)
	assert.Equal(t, 0, result.Deleted)
	assert.Len(t, st.secrets, 2)

	// Simulate the file being deleted from the repo (e.g. removed and
	// pushed) before the next sync.
	require.NoError(t, os.Remove(filepath.Join(dir, "secrets/ns/svc/remove.yaml")))

	// Second sync: same repoID. The removed file must be detected and its
	// secret deleted from the store; the untouched one must survive.
	result, err = syncer.SyncFromDir(ctx, dir, "secrets/", "sha2", "repo-1")
	require.NoError(t, err)
	assert.Equal(t, 1, result.Deleted, "the secret whose file was removed must be deleted")
	assert.Len(t, st.secrets, 1)
	_, stillPresent := st.secrets[secretKey("ns", "svc", "keep")]
	assert.True(t, stillPresent, "the secret whose file was NOT removed must survive")
	_, removed := st.secrets[secretKey("ns", "svc", "remove")]
	assert.False(t, removed, "the secret whose file WAS removed must be gone")
}

// TestSyncFromDir_EmptyRepoIDSkipsDeletionDetection verifies that SyncBundle's
// use case (no registered repository, repoID == "") never deletes anything,
// even across two syncs where a file disappears — there is no reliable
// "previous sync" state to diff against without a repo to scope it to.
func TestSyncFromDir_EmptyRepoIDSkipsDeletionDetection(t *testing.T) {
	dir := t.TempDir()
	keys := &mockKeys{}
	st, pubKey := newSyncableStore(t, keys)
	sopsEncrypt(t, dir, "secrets/ns/svc/only.yaml", "value: only\n", pubKey)

	syncer := NewSyncer(st, keys, nil, "")
	ctx := context.Background()

	result, err := syncer.SyncFromDir(ctx, dir, "secrets/", "sha1", "")
	require.NoError(t, err)
	assert.Equal(t, 1, result.Added)

	require.NoError(t, os.Remove(filepath.Join(dir, "secrets/ns/svc/only.yaml")))

	result, err = syncer.SyncFromDir(ctx, dir, "secrets/", "sha2", "")
	require.NoError(t, err)
	assert.Equal(t, 0, result.Deleted, "repoID=\"\" must never trigger deletion detection")
	assert.Len(t, st.secrets, 1, "the secret must still be present since deletion detection was skipped")
}

// TestSyncFromDir_DoesNotDeleteAnotherRepoSSecrets verifies that deletion
// detection is scoped per-repo: a secret attributed to a different repo_id
// must never be touched by another repo's sync, even if that other repo's
// walk doesn't happen to include it.
func TestSyncFromDir_DoesNotDeleteAnotherRepoSSecrets(t *testing.T) {
	keys := &mockKeys{}
	st, pubKey := newSyncableStore(t, keys)

	dirA := t.TempDir()
	sopsEncrypt(t, dirA, "secrets/ns/svc/a.yaml", "value: a\n", pubKey)

	dirB := t.TempDir()
	sopsEncrypt(t, dirB, "secrets/ns/svc/b.yaml", "value: b\n", pubKey)

	syncer := NewSyncer(st, keys, nil, "")
	ctx := context.Background()

	_, err := syncer.SyncFromDir(ctx, dirA, "secrets/", "sha1", "repo-a")
	require.NoError(t, err)
	_, err = syncer.SyncFromDir(ctx, dirB, "secrets/", "sha1", "repo-b")
	require.NoError(t, err)
	require.Len(t, st.secrets, 2)

	// Re-sync repo-a; repo-b's secret must survive untouched even though
	// repo-a's walk never sees it.
	result, err := syncer.SyncFromDir(ctx, dirA, "secrets/", "sha2", "repo-a")
	require.NoError(t, err)
	assert.Equal(t, 0, result.Deleted)
	assert.Len(t, st.secrets, 2, "repo-b's secret must not be affected by repo-a's sync")
}

// TestSyncConfigFromDir_DeletesConfigRemovedFromRepo mirrors the secret
// deletion-tracking test above for the plain-YAML config path.
func TestSyncConfigFromDir_DeletesConfigRemovedFromRepo(t *testing.T) {
	dir := t.TempDir()
	keepDir := filepath.Join(dir, "config", "ns-keep")
	require.NoError(t, os.MkdirAll(keepDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(keepDir, "svc.yaml"), []byte("port: 1\n"), 0o600))

	removeDir := filepath.Join(dir, "config", "ns-remove")
	require.NoError(t, os.MkdirAll(removeDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(removeDir, "svc.yaml"), []byte("port: 2\n"), 0o600))

	st := &statefulKEKStore{}
	syncer := NewSyncer(st, &mockKeys{}, nil, "")
	ctx := context.Background()

	count, deleted, err := syncer.SyncConfigFromDir(ctx, dir, "config/", "repo-1")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.Equal(t, 0, deleted)

	require.NoError(t, os.Remove(filepath.Join(removeDir, "svc.yaml")))

	count, deleted, err = syncer.SyncConfigFromDir(ctx, dir, "config/", "repo-1")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, 1, deleted)
	_, stillPresent := st.configs[configKey("ns-keep", "svc")]
	assert.True(t, stillPresent)
	_, removed := st.configs[configKey("ns-remove", "svc")]
	assert.False(t, removed)
}
