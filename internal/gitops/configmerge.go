package gitops

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch/v5"
)

// mergeConfigSync is the merge decision function passed to
// store.SyncServiceConfig — see its doc comment for the overall contract.
// It implements a 3-way merge (bytepunx/signet#45) so a git sync can never
// silently revert a live PatchServiceConfig write: synced is the config's
// content as of the last git sync (the merge base), live is the config's
// current content (possibly patched since), and git is what the sync just
// read from the repository.
//
// Both sides are diffed against synced using RFC 7396 merge-patch semantics
// (jsonpatch.CreateMergePatch) to find which JSON paths changed:
//   - If git has no changes relative to synced, skip — nothing to do.
//   - If live has no changes relative to synced (no patch happened since
//     the last sync), fast-forward to git's content.
//   - If the two sides changed disjoint sets of paths (e.g. git edited
//     tenants.acme, a patch touched tenants.other), auto-merge: apply
//     git's delta on top of the live content. Neither side's change is
//     lost.
//   - If the two sides touched any of the same paths, this is a genuine
//     conflict — signet does not silently pick a winner.
func mergeConfigSync(synced, live, git json.RawMessage) (newContent json.RawMessage, conflict bool, err error) {
	if len(synced) == 0 {
		// No baseline recorded yet (config predates the synced_content
		// column, or this is its first sync from git) — there's nothing
		// safe to diff against, so git wins outright, same as the
		// unconditional-overwrite behavior this fix replaces. This
		// establishes a baseline; every subsequent sync uses the full
		// 3-way merge below.
		return git, false, nil
	}

	gitPatch, err := jsonpatch.CreateMergePatch(synced, git)
	if err != nil {
		return nil, false, fmt.Errorf("diff git content against last-synced baseline: %w", err)
	}
	gitPaths := changedPaths(gitPatch)
	if len(gitPaths) == 0 {
		return nil, false, nil
	}

	livePatch, err := jsonpatch.CreateMergePatch(synced, live)
	if err != nil {
		return nil, false, fmt.Errorf("diff live content against last-synced baseline: %w", err)
	}
	livePaths := changedPaths(livePatch)
	if len(livePaths) == 0 {
		return git, false, nil
	}

	for p := range gitPaths {
		if livePaths[p] {
			return nil, true, nil
		}
	}

	merged, err := jsonpatch.MergePatch(live, gitPatch)
	if err != nil {
		return nil, false, fmt.Errorf("apply git-side changes onto live content: %w", err)
	}
	return merged, false, nil
}

// changedPaths flattens a merge-patch document (as produced by
// jsonpatch.CreateMergePatch) into a set of dot-joined leaf paths that
// changed, e.g. {"tenants.acme"} rather than just {"tenants"}. Recursing
// into every nested object is safe: CreateMergePatch's getDiff only leaves
// a nested object at a key when both the original and modified documents
// had an object there (any other change — type mismatch, scalar, array,
// or deletion — is emitted as a leaf value at that key), so a value that
// successfully unmarshals into a non-empty object always represents a
// further, deeper diff rather than a literal replacement value.
func changedPaths(mergePatch json.RawMessage) map[string]bool {
	paths := make(map[string]bool)
	collectChangedPaths(mergePatch, "", paths)
	return paths
}

func collectChangedPaths(mergePatch json.RawMessage, prefix string, out map[string]bool) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(mergePatch, &m); err != nil || len(m) == 0 {
		if prefix != "" {
			out[prefix] = true
		}
		return
	}
	for k, v := range m {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		var nested map[string]json.RawMessage
		if json.Unmarshal(v, &nested) == nil && len(nested) > 0 {
			collectChangedPaths(v, path, out)
		} else {
			out[path] = true
		}
	}
}
