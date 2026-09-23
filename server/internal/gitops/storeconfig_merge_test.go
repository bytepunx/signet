package gitops

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStoreConfig_PatchSurvivesUnchangedResync is the regression test for
// bytepunx/signet#45: PatchServiceConfig's writes used to be silently
// reverted by the next scheduled sync, because storeConfig unconditionally
// overwrote the live content with whatever was in git, even when git's file
// hadn't changed at all since the last sync.
func TestStoreConfig_PatchSurvivesUnchangedResync(t *testing.T) {
	st := &statefulKEKStore{}
	syncer := NewSyncer(st, &mockKeys{}, nil, nil, "")
	ctx := context.Background()

	yaml := []byte("tenants:\n  acme:\n    gens:\n    - 1\n")

	// First sync: establishes the config and its synced_content baseline.
	conflict, err := syncer.storeConfig(ctx, "authstar", "portcullis", yaml, "repo-1", "actor", false)
	require.NoError(t, err)
	require.False(t, conflict)

	// Simulate a PatchServiceConfig call: it writes `content` directly
	// (never through storeConfig/SyncServiceConfig) and never touches
	// synced_content — see store.PatchServiceConfig's doc comment.
	st.configs[configKey("authstar", "portcullis")] = []byte(`{"tenants":{"acme":{"gens":[1,2]}}}`)

	// Next reconciler tick: same git file, unchanged.
	conflict, err = syncer.storeConfig(ctx, "authstar", "portcullis", yaml, "repo-1", "actor", false)
	require.NoError(t, err)
	assert.False(t, conflict)

	assert.JSONEq(t, `{"tenants":{"acme":{"gens":[1,2]}}}`, string(st.configs[configKey("authstar", "portcullis")]),
		"the live patch must survive a resync of unchanged git content")
}

// TestStoreConfig_DisjointGitEditAutoMergesWithLivePatch verifies that a
// real git-side edit to a DIFFERENT part of the document than a pending
// patch touched is auto-merged rather than treated as a conflict — this is
// the exact multi-tenant scenario #38 was built to solve, extended to a
// concurrent git-side edit instead of a concurrent API caller.
func TestStoreConfig_DisjointGitEditAutoMergesWithLivePatch(t *testing.T) {
	st := &statefulKEKStore{}
	syncer := NewSyncer(st, &mockKeys{}, nil, nil, "")
	ctx := context.Background()

	yaml1 := []byte("tenants:\n  acme:\n    gens:\n    - 1\n  other:\n    gens:\n    - 1\n")
	conflict, err := syncer.storeConfig(ctx, "authstar", "portcullis", yaml1, "repo-1", "actor", false)
	require.NoError(t, err)
	require.False(t, conflict)

	// A patch touches "acme" only.
	st.configs[configKey("authstar", "portcullis")] = []byte(`{"tenants":{"acme":{"gens":[1,2]},"other":{"gens":[1]}}}`)

	// Git is re-synced with an edit to "other" only.
	yaml2 := []byte("tenants:\n  acme:\n    gens:\n    - 1\n  other:\n    gens:\n    - 1\n    - 2\n")
	conflict, err = syncer.storeConfig(ctx, "authstar", "portcullis", yaml2, "repo-1", "actor", false)
	require.NoError(t, err)
	assert.False(t, conflict)

	assert.JSONEq(t, `{"tenants":{"acme":{"gens":[1,2]},"other":{"gens":[1,2]}}}`,
		string(st.configs[configKey("authstar", "portcullis")]),
		"the patch to acme and the git-side edit to other must both survive")
}

// TestStoreConfig_OverlappingGitEditConflicts verifies that a git-side edit
// to the SAME field a pending patch touched is flagged as a conflict, not
// silently resolved either way.
func TestStoreConfig_OverlappingGitEditConflicts(t *testing.T) {
	st := &statefulKEKStore{}
	syncer := NewSyncer(st, &mockKeys{}, nil, nil, "")
	ctx := context.Background()

	yaml1 := []byte("tenants:\n  acme:\n    gens:\n    - 1\n")
	conflict, err := syncer.storeConfig(ctx, "authstar", "portcullis", yaml1, "repo-1", "actor", false)
	require.NoError(t, err)
	require.False(t, conflict)

	st.configs[configKey("authstar", "portcullis")] = []byte(`{"tenants":{"acme":{"gens":[1,2]}}}`)

	// Git independently changed the SAME field.
	yaml2 := []byte("tenants:\n  acme:\n    gens:\n    - 1\n    - 3\n")
	conflict, err = syncer.storeConfig(ctx, "authstar", "portcullis", yaml2, "repo-1", "actor", false)
	require.NoError(t, err)
	assert.True(t, conflict)

	assert.JSONEq(t, `{"tenants":{"acme":{"gens":[1,2]}}}`, string(st.configs[configKey("authstar", "portcullis")]),
		"a genuine conflict must leave the live content untouched, not silently pick a winner")
}

// TestStoreConfig_ForceTakesGitOnConflict verifies TriggerSync's force flag
// resolves a conflict by discarding the live patch and taking git's version.
func TestStoreConfig_ForceTakesGitOnConflict(t *testing.T) {
	st := &statefulKEKStore{}
	syncer := NewSyncer(st, &mockKeys{}, nil, nil, "")
	ctx := context.Background()

	yaml1 := []byte("tenants:\n  acme:\n    gens:\n    - 1\n")
	_, err := syncer.storeConfig(ctx, "authstar", "portcullis", yaml1, "repo-1", "actor", false)
	require.NoError(t, err)

	st.configs[configKey("authstar", "portcullis")] = []byte(`{"tenants":{"acme":{"gens":[1,2]}}}`)

	yaml2 := []byte("tenants:\n  acme:\n    gens:\n    - 1\n    - 3\n")
	conflict, err := syncer.storeConfig(ctx, "authstar", "portcullis", yaml2, "repo-1", "actor", true)
	require.NoError(t, err)
	assert.False(t, conflict, "force must resolve the conflict rather than report it")
	assert.JSONEq(t, `{"tenants":{"acme":{"gens":[1,3]}}}`, string(st.configs[configKey("authstar", "portcullis")]),
		"force must take git's version, discarding the unreflected patch")
}
