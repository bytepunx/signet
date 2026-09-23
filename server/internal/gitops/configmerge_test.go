package gitops

import (
	"encoding/json"
	"testing"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeConfigSync_NoBaseline_GitWinsOutright(t *testing.T) {
	git := json.RawMessage(`{"a":1}`)
	got, conflict, err := mergeConfigSync(nil, json.RawMessage(`{"a":2}`), git)
	require.NoError(t, err)
	assert.False(t, conflict)
	assert.JSONEq(t, string(git), string(got))
}

func TestMergeConfigSync_GitUnchanged_Skips(t *testing.T) {
	synced := json.RawMessage(`{"tenants":{"acme":{"gens":[1]}}}`)
	live := json.RawMessage(`{"tenants":{"acme":{"gens":[1,2]}}}`) // patched since last sync
	got, conflict, err := mergeConfigSync(synced, live, synced)
	require.NoError(t, err)
	assert.False(t, conflict)
	assert.Nil(t, got, "git content identical to the last-synced baseline must not overwrite a live patch")
}

func TestMergeConfigSync_LiveUnchanged_FastForwards(t *testing.T) {
	synced := json.RawMessage(`{"port":8080}`)
	live := synced // no patch since last sync
	git := json.RawMessage(`{"port":9090}`)
	got, conflict, err := mergeConfigSync(synced, live, git)
	require.NoError(t, err)
	assert.False(t, conflict)
	assert.JSONEq(t, string(git), string(got))
}

func TestMergeConfigSync_DisjointChanges_AutoMerges(t *testing.T) {
	synced := json.RawMessage(`{"tenants":{"acme":{"gens":[1]},"other":{"gens":[1]}}}`)
	// git edited "other"; live (via PatchServiceConfig) edited "acme".
	git := json.RawMessage(`{"tenants":{"acme":{"gens":[1]},"other":{"gens":[1,2]}}}`)
	live := json.RawMessage(`{"tenants":{"acme":{"gens":[1,2]},"other":{"gens":[1]}}}`)

	got, conflict, err := mergeConfigSync(synced, live, git)
	require.NoError(t, err)
	require.False(t, conflict)
	assert.JSONEq(t, `{"tenants":{"acme":{"gens":[1,2]},"other":{"gens":[1,2]}}}`, string(got),
		"neither side's change should be lost")
}

func TestMergeConfigSync_OverlappingChanges_Conflicts(t *testing.T) {
	synced := json.RawMessage(`{"tenants":{"acme":{"gens":[1]}}}`)
	git := json.RawMessage(`{"tenants":{"acme":{"gens":[1,2]}}}`)
	live := json.RawMessage(`{"tenants":{"acme":{"gens":[1,3]}}}`)

	got, conflict, err := mergeConfigSync(synced, live, git)
	require.NoError(t, err)
	assert.True(t, conflict)
	assert.Nil(t, got, "a genuine conflict must never silently pick a winner")
}

func TestMergeConfigSync_TopLevelDisjointKeys_AutoMerges(t *testing.T) {
	synced := json.RawMessage(`{"a":1,"b":2}`)
	git := json.RawMessage(`{"a":10,"b":2}`)
	live := json.RawMessage(`{"a":1,"b":20}`)

	got, conflict, err := mergeConfigSync(synced, live, git)
	require.NoError(t, err)
	require.False(t, conflict)
	assert.JSONEq(t, `{"a":10,"b":20}`, string(got))
}

func TestMergeConfigSync_BothSidesNoop_NoWrite(t *testing.T) {
	synced := json.RawMessage(`{"a":1}`)
	got, conflict, err := mergeConfigSync(synced, synced, synced)
	require.NoError(t, err)
	assert.False(t, conflict)
	assert.Nil(t, got)
}

func TestChangedPaths_NestedObjectRecursesToLeaf(t *testing.T) {
	patch, err := marshalMergePatch(t,
		`{"tenants":{"acme":{"gens":[1]},"other":{"gens":[1]}}}`,
		`{"tenants":{"acme":{"gens":[1,2]},"other":{"gens":[1]}}}`)
	require.NoError(t, err)
	paths := changedPaths(patch)
	assert.Equal(t, map[string]bool{"tenants.acme.gens": true}, paths,
		"gens is an array, not an object, so recursion continues past tenants.acme down to the array field itself")
}

func TestChangedPaths_KeyDeletionIsALeafChange(t *testing.T) {
	patch, err := marshalMergePatch(t, `{"a":1,"b":2}`, `{"a":1}`)
	require.NoError(t, err)
	paths := changedPaths(patch)
	assert.Equal(t, map[string]bool{"b": true}, paths)
}

func TestChangedPaths_TypeChangeIsALeafChange(t *testing.T) {
	// "tenants" goes from an object to a string — a type mismatch, so it
	// must register as a single leaf change at "tenants", not recurse.
	patch, err := marshalMergePatch(t, `{"tenants":{"acme":1}}`, `{"tenants":"disabled"}`)
	require.NoError(t, err)
	paths := changedPaths(patch)
	assert.Equal(t, map[string]bool{"tenants": true}, paths)
}

func TestChangedPaths_NoDifference_EmptySet(t *testing.T) {
	patch, err := marshalMergePatch(t, `{"a":1}`, `{"a":1}`)
	require.NoError(t, err)
	assert.Empty(t, changedPaths(patch))
}

// marshalMergePatch is a small test helper wrapping jsonpatch.CreateMergePatch
// so the changedPaths tests above can be written directly against
// before/after JSON literals instead of hand-authored merge-patch bytes.
func marshalMergePatch(t *testing.T, original, modified string) (json.RawMessage, error) {
	t.Helper()
	return jsonpatch.CreateMergePatch([]byte(original), []byte(modified))
}
