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
