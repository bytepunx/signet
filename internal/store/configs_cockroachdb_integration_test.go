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
