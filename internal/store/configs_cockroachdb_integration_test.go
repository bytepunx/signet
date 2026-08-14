//go:build crdb_integration

package store

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPatchServiceConfig_ConcurrentPatchesNeverLoseAnUpdate is the
// regression test for bytepunx/signet#38's actual design requirement: two
// concurrent PatchServiceConfig calls against the same (namespace, service)
// must never silently drop one caller's update the way a client-side
// read-modify-write would. This requires real CockroachDB — its always-on
// serializable isolation is what guarantees one of the two transactions
// aborts with ErrConflict rather than blindly overwriting the other; a
// Postgres-backed test (the "integration" suite) would not exercise this
// property faithfully (see this package's crdb_integration file header).
func TestPatchServiceConfig_ConcurrentPatchesNeverLoseAnUpdate(t *testing.T) {
	s := newCRDBTestStore(t)
	ctx := context.Background()

	_, err := s.pool.Exec(ctx, "DELETE FROM configs")
	require.NoError(t, err)
	require.NoError(t, s.PutServiceConfig(ctx, "authstar", "portcullis",
		json.RawMessage(`{"tenants":{"acme":{"gens":[]},"other":{"gens":[]}}}`), ""))

	appendGen := func(tenant string, gen float64) func(current json.RawMessage) (json.RawMessage, error) {
		return func(current json.RawMessage) (json.RawMessage, error) {
			var doc map[string]any
			if err := json.Unmarshal(current, &doc); err != nil {
				return nil, err
			}
			tenants := doc["tenants"].(map[string]any)
			entry := tenants[tenant].(map[string]any)
			gens := entry["gens"].([]any)
			entry["gens"] = append(gens, gen)
			return json.Marshal(doc)
		}
	}

	// patchWithRetry mirrors what a real caller (the API-layer
	// PatchServiceConfig RPC handler) is expected to do: retry the whole
	// operation, including re-deriving the patch from fresh state, on
	// ErrConflict. The store method itself does not retry internally.
	patchWithRetry := func(tenant string, gen float64) {
		for {
			_, err := s.PatchServiceConfig(ctx, "authstar", "portcullis", appendGen(tenant, gen))
			if err == nil {
				return
			}
			if errors.Is(err, ErrConflict) {
				continue
			}
			require.NoError(t, err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); patchWithRetry("acme", 1) }()
	go func() { defer wg.Done(); patchWithRetry("other", 1) }()
	wg.Wait()

	content, _, err := s.GetServiceConfig(ctx, "authstar", "portcullis")
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(content, &doc))
	tenants := doc["tenants"].(map[string]any)
	assert.Equal(t, []any{float64(1)}, tenants["acme"].(map[string]any)["gens"],
		"acme's concurrent update must not have been lost")
	assert.Equal(t, []any{float64(1)}, tenants["other"].(map[string]any)["gens"],
		"other's concurrent update must not have been lost")
}

// TestSyncServiceConfig_ConcurrentWithPatchServiceConfig_NeverLosesAnUpdate
// is the regression test for bytepunx/signet#45's actual risk: unlike #38's
// test above (two concurrent callers of the SAME method), this races two
// DIFFERENT write paths — an API-driven PatchServiceConfig call and a
// git-sync-driven SyncServiceConfig call — against the same row, the exact
// scenario that used to let a sync silently revert a patch. Real
// CockroachDB is required for the same reason as the test above: its
// always-on serializable isolation is what guarantees one of the two
// transactions aborts with ErrConflict rather than either blindly
// overwriting the other.
func TestSyncServiceConfig_ConcurrentWithPatchServiceConfig_NeverLosesAnUpdate(t *testing.T) {
	s := newCRDBTestStore(t)
	ctx := context.Background()

	_, err := s.pool.Exec(ctx, "DELETE FROM configs")
	require.NoError(t, err)

	// Seed via SyncServiceConfig so synced_content starts populated,
	// establishing a real merge baseline.
	_, _, err = s.SyncServiceConfig(ctx, "authstar", "portcullis",
		json.RawMessage(`{"tenants":{"acme":{"gens":[1]},"other":{"gens":[1]}}}`), "",
		func(_, _, _ json.RawMessage) (json.RawMessage, bool, error) { return nil, false, nil })
	require.NoError(t, err)

	appendAcmeGen := func(current json.RawMessage) (json.RawMessage, error) {
		var doc map[string]any
		if err := json.Unmarshal(current, &doc); err != nil {
			return nil, err
		}
		entry := doc["tenants"].(map[string]any)["acme"].(map[string]any)
		entry["gens"] = append(entry["gens"].([]any), float64(2))
		return json.Marshal(doc)
	}
	patchWithRetry := func() {
		for {
			_, err := s.PatchServiceConfig(ctx, "authstar", "portcullis", appendAcmeGen)
			if err == nil {
				return
			}
			if errors.Is(err, ErrConflict) {
				continue
			}
			require.NoError(t, err)
		}
	}

	// Simulates a git sync that independently changed "other" — a disjoint
	// merge, mirroring gitops.mergeConfigSync's auto-merge case (not
	// importable here: internal/gitops imports internal/store).
	newOtherContent := json.RawMessage(`{"tenants":{"acme":{"gens":[1]},"other":{"gens":[1,2]}}}`)
	syncWithRetry := func() {
		for {
			merge := func(_, live, _ json.RawMessage) (json.RawMessage, bool, error) {
				// Re-derive from the fresh live content each attempt, the
				// same way a real retry must — see PatchServiceConfig's
				// doc comment on retrying the whole operation, not just
				// the commit.
				var liveDoc map[string]any
				if err := json.Unmarshal(live, &liveDoc); err != nil {
					return nil, false, err
				}
				var otherDoc map[string]any
				require.NoError(t, json.Unmarshal(newOtherContent, &otherDoc))
				liveDoc["tenants"].(map[string]any)["other"] = otherDoc["tenants"].(map[string]any)["other"]
				merged, err := json.Marshal(liveDoc)
				return merged, false, err
			}
			_, conflict, err := s.SyncServiceConfig(ctx, "authstar", "portcullis", newOtherContent, "", merge)
			if err != nil {
				if errors.Is(err, ErrConflict) {
					continue
				}
				require.NoError(t, err)
			}
			if conflict {
				continue // shouldn't happen for a disjoint change, but retry rather than hang if it does
			}
			return
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); patchWithRetry() }()
	go func() { defer wg.Done(); syncWithRetry() }()
	wg.Wait()

	content, _, err := s.GetServiceConfig(ctx, "authstar", "portcullis")
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(content, &doc))
	tenants := doc["tenants"].(map[string]any)
	assert.Equal(t, []any{float64(1), float64(2)}, tenants["acme"].(map[string]any)["gens"],
		"the PatchServiceConfig update to acme must not have been lost")
	assert.Equal(t, []any{float64(1), float64(2)}, tenants["other"].(map[string]any)["gens"],
		"the SyncServiceConfig update to other must not have been lost")
}
