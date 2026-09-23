//go:build integration

package store

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cleanConfigs(t *testing.T, s *Store) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), "DELETE FROM configs")
	require.NoError(t, err)
}

// TestPatchServiceConfig_AppliesAndReturnsNewVersion is the basic
// happy-path regression test for bytepunx/signet#38: apply's return value
// becomes the new stored document, and the version increments.
func TestPatchServiceConfig_AppliesAndReturnsNewVersion(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	require.NoError(t, s.PutServiceConfig(ctx, "authstar", "portcullis",
		json.RawMessage(`{"tenants":{"acme":{"sessionKeyGenerations":[1]}}}`), ""))

	version, err := s.PatchServiceConfig(ctx, "authstar", "portcullis", func(current json.RawMessage) (json.RawMessage, error) {
		var doc map[string]any
		require.NoError(t, json.Unmarshal(current, &doc))
		tenants := doc["tenants"].(map[string]any)
		acme := tenants["acme"].(map[string]any)
		gens := acme["sessionKeyGenerations"].([]any)
		acme["sessionKeyGenerations"] = append(gens, float64(2))
		return json.Marshal(doc)
	})
	require.NoError(t, err)
	assert.Equal(t, 2, version, "version must increment from the PutServiceConfig-created row")

	content, gotVersion, err := s.GetServiceConfig(ctx, "authstar", "portcullis")
	require.NoError(t, err)
	assert.Equal(t, 2, gotVersion)
	assert.JSONEq(t, `{"tenants":{"acme":{"sessionKeyGenerations":[1,2]}}}`, string(content))
}

// TestPatchServiceConfig_NotFoundWhenNoConfigExists verifies this method
// only mutates an existing document — it must never silently create one,
// unlike PutServiceConfig.
func TestPatchServiceConfig_NotFoundWhenNoConfigExists(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	called := false
	_, err := s.PatchServiceConfig(ctx, "ns-missing", "svc-missing", func(current json.RawMessage) (json.RawMessage, error) {
		called = true
		return current, nil
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.False(t, called, "apply must never be invoked when there's nothing to patch")
}

// TestPatchServiceConfig_ApplyErrorRollsBackNoWrite verifies a failing
// apply (e.g. a malformed JSON Patch, at the API layer) leaves the stored
// document completely untouched — the transaction must roll back, not
// partially commit.
func TestPatchServiceConfig_ApplyErrorRollsBackNoWrite(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	require.NoError(t, s.PutServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"a":1}`), ""))

	applyErr := errors.New("boom")
	_, err := s.PatchServiceConfig(ctx, "ns", "svc", func(current json.RawMessage) (json.RawMessage, error) {
		return nil, applyErr
	})
	require.ErrorIs(t, err, applyErr)

	content, version, err := s.GetServiceConfig(ctx, "ns", "svc")
	require.NoError(t, err)
	assert.Equal(t, 1, version, "version must not have incremented")
	assert.JSONEq(t, `{"a":1}`, string(content))
}

// --- SyncServiceConfig tests (bytepunx/signet#45) ---

// TestSyncServiceConfig_FirstWriteInsertsAndSeedsBaseline verifies that
// syncing a namespace/service with no existing config inserts it directly,
// as both the live content and the synced_content baseline — merge must
// never be called since there's nothing to reconcile against yet.
func TestSyncServiceConfig_FirstWriteInsertsAndSeedsBaseline(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	called := false
	version, conflict, err := s.SyncServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"a":1}`), "",
		func(_, _, _ json.RawMessage) (json.RawMessage, bool, error) {
			called = true
			return nil, false, nil
		})
	require.NoError(t, err)
	assert.Equal(t, 1, version)
	assert.False(t, conflict)
	assert.False(t, called, "merge must not be invoked when there's no existing row to reconcile against")

	content, gotVersion, err := s.GetServiceConfig(ctx, "ns", "svc")
	require.NoError(t, err)
	assert.Equal(t, 1, gotVersion)
	assert.JSONEq(t, `{"a":1}`, string(content))
}

// TestSyncServiceConfig_MergeSkip_NoWrite verifies that merge returning
// (nil, false, nil) — "nothing changed" — leaves the row completely
// untouched, including its version.
func TestSyncServiceConfig_MergeSkip_NoWrite(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	require.NoError(t, s.PutServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"a":1}`), ""))

	version, conflict, err := s.SyncServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"a":1}`), "",
		func(_, _, _ json.RawMessage) (json.RawMessage, bool, error) {
			return nil, false, nil
		})
	require.NoError(t, err)
	assert.Equal(t, 0, version, "no write happened, so there's no new version to report")
	assert.False(t, conflict)

	content, gotVersion, err := s.GetServiceConfig(ctx, "ns", "svc")
	require.NoError(t, err)
	assert.Equal(t, 1, gotVersion, "version must not have incremented")
	assert.JSONEq(t, `{"a":1}`, string(content))
}

// TestSyncServiceConfig_MergeWrite_UpdatesContentAndSyncedContent verifies
// that merge returning new content writes it as the live content AND
// advances synced_content to gitContent — checked directly via SQL since
// synced_content isn't exposed through GetServiceConfig.
func TestSyncServiceConfig_MergeWrite_UpdatesContentAndSyncedContent(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	// Seed via SyncServiceConfig itself so synced_content starts populated
	// (PutServiceConfig never sets it — see its own tests above).
	_, _, err := s.SyncServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"a":1}`), "",
		func(_, _, _ json.RawMessage) (json.RawMessage, bool, error) {
			t.Fatal("merge must not be invoked on first write")
			return nil, false, nil
		})
	require.NoError(t, err)

	version, conflict, err := s.SyncServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"a":2}`), "",
		func(synced, live, git json.RawMessage) (json.RawMessage, bool, error) {
			assert.JSONEq(t, `{"a":1}`, string(synced))
			assert.JSONEq(t, `{"a":1}`, string(live))
			assert.JSONEq(t, `{"a":2}`, string(git))
			return json.RawMessage(`{"a":"merged"}`), false, nil
		})
	require.NoError(t, err)
	assert.Equal(t, 2, version)
	assert.False(t, conflict)

	content, gotVersion, err := s.GetServiceConfig(ctx, "ns", "svc")
	require.NoError(t, err)
	assert.Equal(t, 2, gotVersion)
	assert.JSONEq(t, `{"a":"merged"}`, string(content))

	var syncedContent []byte
	require.NoError(t, s.pool.QueryRow(ctx,
		"SELECT synced_content FROM configs WHERE namespace = $1 AND service = $2", "ns", "svc",
	).Scan(&syncedContent))
	assert.JSONEq(t, `{"a":2}`, string(syncedContent), "synced_content must advance to gitContent, not the merged result")
}

// TestSyncServiceConfig_Conflict_NoWriteNoBaselineAdvance verifies that a
// reported conflict leaves both content and synced_content completely
// untouched, so the same conflict resurfaces on the next sync rather than
// silently resolving.
func TestSyncServiceConfig_Conflict_NoWriteNoBaselineAdvance(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	require.NoError(t, s.PutServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"a":1}`), ""))

	version, conflict, err := s.SyncServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"a":2}`), "",
		func(_, _, _ json.RawMessage) (json.RawMessage, bool, error) {
			return nil, true, nil
		})
	require.NoError(t, err)
	assert.Equal(t, 0, version)
	assert.True(t, conflict)

	content, gotVersion, err := s.GetServiceConfig(ctx, "ns", "svc")
	require.NoError(t, err)
	assert.Equal(t, 1, gotVersion, "version must not have incremented")
	assert.JSONEq(t, `{"a":1}`, string(content))

	var syncedContent []byte
	require.NoError(t, s.pool.QueryRow(ctx,
		"SELECT synced_content FROM configs WHERE namespace = $1 AND service = $2", "ns", "svc",
	).Scan(&syncedContent))
	assert.Nil(t, syncedContent, "a config never previously synced via SyncServiceConfig has no baseline to preserve")
}

// TestSyncServiceConfig_MergeErrorRollsBackNoWrite verifies a failing merge
// (an error, not a conflict) leaves the row untouched, same as
// PatchServiceConfig's apply-error contract.
func TestSyncServiceConfig_MergeErrorRollsBackNoWrite(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	require.NoError(t, s.PutServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"a":1}`), ""))

	mergeErr := errors.New("boom")
	_, _, err := s.SyncServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"a":2}`), "",
		func(_, _, _ json.RawMessage) (json.RawMessage, bool, error) {
			return nil, false, mergeErr
		})
	require.ErrorIs(t, err, mergeErr)

	content, version, err := s.GetServiceConfig(ctx, "ns", "svc")
	require.NoError(t, err)
	assert.Equal(t, 1, version)
	assert.JSONEq(t, `{"a":1}`, string(content))
}

// --- PutServiceConfigIfVersion tests (bytepunx/signet#80) ---

// TestPutServiceConfigIfVersion_CreateOnlySuccess verifies expectedVersion=0
// creates a brand-new document when none exists, leaving repo_id and
// synced_content both NULL — this method is deliberately git-free.
func TestPutServiceConfigIfVersion_CreateOnlySuccess(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	version, err := s.PutServiceConfigIfVersion(ctx, "ns", "svc", json.RawMessage(`{"a":1}`), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, version)

	content, gotVersion, err := s.GetServiceConfig(ctx, "ns", "svc")
	require.NoError(t, err)
	assert.Equal(t, 1, gotVersion)
	assert.JSONEq(t, `{"a":1}`, string(content))

	var repoID, syncedContent []byte
	require.NoError(t, s.pool.QueryRow(ctx,
		"SELECT repo_id, synced_content FROM configs WHERE namespace = $1 AND service = $2", "ns", "svc",
	).Scan(&repoID, &syncedContent))
	assert.Nil(t, repoID, "a config created via PutServiceConfigIfVersion has no git provenance")
	assert.Nil(t, syncedContent, "a config created via PutServiceConfigIfVersion has no sync baseline")
}

// TestPutServiceConfigIfVersion_CreateOnlyAgainstExisting_AlreadyExists
// verifies expectedVersion=0 refuses to overwrite an existing document —
// this is the "re-run of a Helm install hook must not stomp config an
// operator has since changed" guarantee.
func TestPutServiceConfigIfVersion_CreateOnlyAgainstExisting_AlreadyExists(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	_, err := s.PutServiceConfigIfVersion(ctx, "ns", "svc", json.RawMessage(`{"a":1}`), 0)
	require.NoError(t, err)

	_, err = s.PutServiceConfigIfVersion(ctx, "ns", "svc", json.RawMessage(`{"a":2}`), 0)
	require.ErrorIs(t, err, ErrAlreadyExists)

	content, version, err := s.GetServiceConfig(ctx, "ns", "svc")
	require.NoError(t, err)
	assert.Equal(t, 1, version, "the rejected create-only call must not have written anything")
	assert.JSONEq(t, `{"a":1}`, string(content))
}

// TestPutServiceConfigIfVersion_ReplaceAtMatchingVersionSucceeds verifies
// the optimistic-concurrency happy path: replacing at the exact version the
// caller last read succeeds and increments the version.
func TestPutServiceConfigIfVersion_ReplaceAtMatchingVersionSucceeds(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	v1, err := s.PutServiceConfigIfVersion(ctx, "ns", "svc", json.RawMessage(`{"a":1}`), 0)
	require.NoError(t, err)
	require.Equal(t, 1, v1)

	v2, err := s.PutServiceConfigIfVersion(ctx, "ns", "svc", json.RawMessage(`{"a":2}`), v1)
	require.NoError(t, err)
	assert.Equal(t, 2, v2)

	content, gotVersion, err := s.GetServiceConfig(ctx, "ns", "svc")
	require.NoError(t, err)
	assert.Equal(t, 2, gotVersion)
	assert.JSONEq(t, `{"a":2}`, string(content))
}

// TestPutServiceConfigIfVersion_ReplaceAtStaleVersion_Conflict verifies a
// caller replacing against a version that no longer matches current state
// (someone else wrote in between) gets ErrConflict, not a silent overwrite
// — the exact failure class this RPC exists to avoid.
func TestPutServiceConfigIfVersion_ReplaceAtStaleVersion_Conflict(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	v1, err := s.PutServiceConfigIfVersion(ctx, "ns", "svc", json.RawMessage(`{"a":1}`), 0)
	require.NoError(t, err)

	// Someone else advances the version in between.
	_, err = s.PutServiceConfigIfVersion(ctx, "ns", "svc", json.RawMessage(`{"a":2}`), v1)
	require.NoError(t, err)

	// The original caller, still holding the stale v1, tries to replace.
	_, err = s.PutServiceConfigIfVersion(ctx, "ns", "svc", json.RawMessage(`{"a":"stale-write"}`), v1)
	require.ErrorIs(t, err, ErrConflict)

	content, gotVersion, err := s.GetServiceConfig(ctx, "ns", "svc")
	require.NoError(t, err)
	assert.Equal(t, 2, gotVersion, "the conflicting call must not have written anything")
	assert.JSONEq(t, `{"a":2}`, string(content))
}

// TestPutServiceConfigIfVersion_ReplaceNonexistent_NotFound verifies a
// nonzero expectedVersion against a document that was never created at all
// fails NotFound, not Conflict — you can't be "replacing" something that
// doesn't exist.
func TestPutServiceConfigIfVersion_ReplaceNonexistent_NotFound(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	_, err := s.PutServiceConfigIfVersion(ctx, "ns-missing", "svc-missing", json.RawMessage(`{"a":1}`), 3)
	require.ErrorIs(t, err, ErrNotFound)
}

// TestPutServiceConfigIfVersion_RejectsNegativeExpectedVersion verifies
// input validation happens before any transaction is opened.
func TestPutServiceConfigIfVersion_RejectsNegativeExpectedVersion(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	_, err := s.PutServiceConfigIfVersion(ctx, "ns", "svc", json.RawMessage(`{"a":1}`), -1)
	require.ErrorIs(t, err, ErrInvalidInput)
}

// TestPutServiceConfigIfVersion_ProtectedByGitSyncMerge verifies bytepunx/signet#45's
// 3-way merge protects a PutServiceConfigIfVersion-created row exactly like
// it already protects PatchServiceConfig's writes: a config created via
// this method (no synced_content baseline) that later gets a registered git
// source treats the first sync as a fresh baseline-seed, not a silent
// overwrite of the live content.
func TestPutServiceConfigIfVersion_ProtectedByGitSyncMerge(t *testing.T) {
	s := newTestStore(t)
	cleanConfigs(t, s)
	ctx := context.Background()

	_, err := s.PutServiceConfigIfVersion(ctx, "ns", "svc", json.RawMessage(`{"a":"from-helm-hook"}`), 0)
	require.NoError(t, err)

	// A git source is registered for this service after the fact and syncs
	// for the first time. SyncServiceConfig must never invoke merge here —
	// there's no synced_content baseline yet, so the existing row's own
	// "first write" contract applies... except the row already exists (from
	// the Helm hook), so this exercises the merge path with an empty
	// baseline rather than the true-first-write INSERT path.
	merged := false
	version, conflict, err := s.SyncServiceConfig(ctx, "ns", "svc", json.RawMessage(`{"a":"from-git"}`), "",
		func(synced, live, git json.RawMessage) (json.RawMessage, bool, error) {
			merged = true
			assert.Nil(t, synced, "no prior sync baseline exists for a PutServiceConfigIfVersion-created row")
			assert.JSONEq(t, `{"a":"from-helm-hook"}`, string(live))
			assert.JSONEq(t, `{"a":"from-git"}`, string(git))
			return git, false, nil
		})
	require.NoError(t, err)
	assert.True(t, merged, "a row with existing live content must go through merge, not a blind first-write insert")
	assert.False(t, conflict)
	assert.Equal(t, 2, version)
}
