//go:build integration

package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPolicyServiceDimMigration_RewritesTwoSegmentPatterns exercises the exact
// SQL transformation from migrations/000007_policy_service_dim.sql directly
// (rather than relying on schema_migrations idempotency, since the real
// migration has already run once against the shared test container by the
// time any test executes). This verifies the regex/array-length logic is
// correct against a real CockroachDB-compatible engine.
func TestPolicyServiceDimMigration_RewritesTwoSegmentPatterns(t *testing.T) {
	s := newTestStore(t)
	cleanPolicies(t, s)

	const rewriteSQL = `
		UPDATE access_policies
		SET secret_pattern = regexp_replace(secret_pattern, '^([^/]*)/([^/].*)$', '\1/*/\2')
		WHERE array_length(regexp_split_to_array(secret_pattern, '/'), 1) = 2`

	cases := []struct {
		name    string
		pattern string
		want    string
	}{
		{"simple two-segment", "prod/db-*", "prod/*/db-*"},
		{"wildcard namespace two-segment", "*/api-key", "*/*/api-key"},
		{"already three-segment left untouched", "prod/api/db-*", "prod/api/db-*"},
		{"four-segment left untouched", "prod/api/db/extra", "prod/api/db/extra"},
	}

	for _, tc := range cases {
		require.NoError(t, s.PutPolicy(context.Background(), &Policy{
			SPIFFEID:    "spiffe://x/" + tc.name,
			Namespace:   "prod",
			Pattern:     tc.pattern,
			Permissions: []string{"get"},
		}))
	}

	_, err := s.pool.Exec(context.Background(), rewriteSQL)
	require.NoError(t, err)

	for _, tc := range cases {
		policies, err := s.GetPoliciesForSPIFFE(context.Background(), "spiffe://x/"+tc.name)
		require.NoError(t, err)
		require.Len(t, policies, 1)
		assert.Equal(t, tc.want, policies[0].Pattern, "pattern %q", tc.pattern)
	}
}
