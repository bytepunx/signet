package gitops

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/bytepunx/signet/internal/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAuditRecorder implements auditRecorder, capturing every entry recorded
// so tests can assert on it. Safe for concurrent use since Syncer makes no
// concurrency guarantee about when Record is called relative to other work.
type fakeAuditRecorder struct {
	mu      sync.Mutex
	entries []audit.Entry
	err     error // returned from every Record call; entries are still captured
}

func (f *fakeAuditRecorder) Record(_ context.Context, e audit.Entry) error {
	f.mu.Lock()
	f.entries = append(f.entries, e)
	f.mu.Unlock()
	return f.err
}

func (f *fakeAuditRecorder) all() []audit.Entry {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]audit.Entry, len(f.entries))
	copy(out, f.entries)
	return out
}

// TestSyncFromDir_RecordsAuditEntryOnWrite is the core regression test for
// bytepunx/signet#25: a secret written via SyncFromDir must produce an audit
// entry attributing the write to the caller-supplied actor — previously,
// GitOps writes were never audited at all.
func TestSyncFromDir_RecordsAuditEntryOnWrite(t *testing.T) {
	dir := t.TempDir()
	keys := &mockKeys{}
	st, pubKey := newSyncableStore(t, keys)
	sopsEncrypt(t, dir, "secrets/authstar/tower/tenant-key.yaml", "value: secret\n", pubKey)

	fa := &fakeAuditRecorder{}
	syncer := NewSyncer(st, keys, nil, fa, "")
	_, err := syncer.SyncFromDir(context.Background(), dir, "secrets/", "sha1", "", "actor-under-test")
	require.NoError(t, err)

	entries := fa.all()
	require.Len(t, entries, 1)
	assert.Equal(t, audit.Entry{
		SPIFFEID:   "actor-under-test",
		Action:     "put_secret",
		Namespace:  "authstar",
		SecretName: "tower/tenant-key",
		Outcome:    "permitted",
	}, entries[0])
}

// TestSyncFromDir_NoAuditEntryForUnchangedResync verifies the dedup
// optimization (isUnchanged) is not treated as a write: resyncing identical
// content must not grow the audit log, since nothing was actually written.
func TestSyncFromDir_NoAuditEntryForUnchangedResync(t *testing.T) {
	dir := t.TempDir()
	keys := &mockKeys{}
	st, pubKey := newSyncableStore(t, keys)
	sopsEncrypt(t, dir, "secrets/ns/svc/stable.yaml", "value: stable\n", pubKey)

	fa := &fakeAuditRecorder{}
	syncer := NewSyncer(st, keys, nil, fa, "")
	ctx := context.Background()

	_, err := syncer.SyncFromDir(ctx, dir, "secrets/", "sha1", "", "actor")
	require.NoError(t, err)
	_, err = syncer.SyncFromDir(ctx, dir, "secrets/", "sha2", "", "actor")
	require.NoError(t, err)

	assert.Len(t, fa.all(), 1, "an unchanged resync must not produce a second audit entry")
}

// TestSyncFromDir_DeletionAudited verifies deletion-detection removals
// (bytepunx/signet's repoID-scoped diff) are audited too, not just additions.
func TestSyncFromDir_DeletionAudited(t *testing.T) {
	dir := t.TempDir()
	keys := &mockKeys{}
	st, pubKey := newSyncableStore(t, keys)
	sopsEncrypt(t, dir, "secrets/ns/svc/remove.yaml", "value: remove-me\n", pubKey)

	fa := &fakeAuditRecorder{}
	syncer := NewSyncer(st, keys, nil, fa, "")
	ctx := context.Background()

	_, err := syncer.SyncFromDir(ctx, dir, "secrets/", "sha1", "repo-1", "actor")
	require.NoError(t, err)
	require.NoError(t, os.Remove(filepath.Join(dir, "secrets", "ns", "svc", "remove.yaml")))

	_, err = syncer.SyncFromDir(ctx, dir, "secrets/", "sha2", "repo-1", "actor")
	require.NoError(t, err)

	entries := fa.all()
	require.Len(t, entries, 2, "one put_secret from the first sync, one delete_secret from the second")
	assert.Equal(t, "put_secret", entries[0].Action)
	assert.Equal(t, "delete_secret", entries[1].Action)
	assert.Equal(t, "actor", entries[1].SPIFFEID)
}

// TestSyncConfigFromDir_RecordsAuditEntryOnWrite mirrors
// TestSyncFromDir_RecordsAuditEntryOnWrite for the config-write path.
func TestSyncConfigFromDir_RecordsAuditEntryOnWrite(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "config", "authstar"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "authstar", "tower.yaml"),
		[]byte("tenants: []\n"), 0o600))

	fa := &fakeAuditRecorder{}
	syncer := NewSyncer(&mockStore{}, &mockKeys{}, nil, fa, "")
	_, _, err := syncer.SyncConfigFromDir(context.Background(), dir, "config/", "", "actor-under-test")
	require.NoError(t, err)

	entries := fa.all()
	require.Len(t, entries, 1)
	assert.Equal(t, audit.Entry{
		SPIFFEID:   "actor-under-test",
		Action:     "put_config",
		Namespace:  "authstar",
		SecretName: "tower/<config>",
		Outcome:    "permitted",
	}, entries[0])
}

// TestNilAuditRecorderDoesNotPanic verifies Syncer's audit dependency is
// safely optional (nil), matching bus's existing nilability — production
// wiring must always pass a real *audit.Writer, but unit tests that don't
// care about auditing shouldn't need a fake for every call site.
func TestNilAuditRecorderDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	keys := &mockKeys{}
	st, pubKey := newSyncableStore(t, keys)
	sopsEncrypt(t, dir, "secrets/ns/svc/key.yaml", "value: v\n", pubKey)

	syncer := NewSyncer(st, keys, nil, nil, "")
	_, err := syncer.SyncFromDir(context.Background(), dir, "secrets/", "sha1", "", "actor")
	require.NoError(t, err)
}
