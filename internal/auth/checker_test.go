package auth

import (
	"context"
	"testing"

	"github.com/bytepunx/signet/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// policyStore is satisfied by *store.Store but also lets us build a minimal
// fake. We define the interface locally so tests don't need a real database.
type policyStore interface {
	GetPoliciesForSPIFFE(ctx context.Context, spiffeID string) ([]store.Policy, error)
}

// fakeStore implements policyStore for testing.
type fakeStore struct {
	policies []store.Policy
	err      error
}

func (f *fakeStore) GetPoliciesForSPIFFE(_ context.Context, spiffeID string) ([]store.Policy, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []store.Policy
	for _, p := range f.policies {
		if p.SPIFFEID == spiffeID {
			out = append(out, p)
		}
	}
	return out, nil
}

// allowViaFake mirrors the full Checker.Allow logic — including the
// exact-match bypass — using a fakeStore so tests don't need a real database.
func allowViaFake(ctx context.Context, fs policyStore, spiffeID, permission, namespace, service, secretName string) error {
	if spiffeID == "" {
		return ErrUnauthenticated
	}
	// Mirror the exact-match bypass from Checker.Allow.
	if spiffeNS, spiffeSA := parseKubeSpiffeID(spiffeID); spiffeNS != "" &&
		spiffeNS == namespace && spiffeSA == service {
		return nil
	}
	policies, err := fs.GetPoliciesForSPIFFE(ctx, spiffeID)
	if err != nil {
		return err
	}
	return evalPolicies(policies, spiffeID, permission, namespace, secretName)
}

// --- tests ---

func TestAllow_EmptySPIFFEID(t *testing.T) {
	err := allowViaFake(context.Background(), &fakeStore{}, "", "get", "ns", "svc", "key")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthenticated)
}

func TestAllow_NoPolicies_NonKubeID(t *testing.T) {
	// A non-Kubernetes SPIFFE ID with no policies is always denied.
	fs := &fakeStore{}
	err := allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "ns", "svc", "key")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestAllow_ExactMatchConvention_NoPolicyNeeded(t *testing.T) {
	// A workload with SPIFFE ID encoding ns=payments, sa=api gets automatic
	// access to any secret in namespace=payments / service=api without a policy.
	fs := &fakeStore{} // no policies stored
	err := allowViaFake(context.Background(), fs,
		"spiffe://cluster.local/ns/payments/sa/api", "get", "payments", "api", "stripe-key")
	require.NoError(t, err)
}

func TestAllow_ExactMatchConvention_AnySecretName(t *testing.T) {
	// The bypass applies regardless of the secret name — any secret under
	// the matching namespace/service is implicitly accessible.
	fs := &fakeStore{}
	spiffeID := "spiffe://cluster.local/ns/infra/sa/redis"
	for _, name := range []string{"password", "tls-cert", "replication-key"} {
		assert.NoError(t, allowViaFake(context.Background(), fs, spiffeID, "get", "infra", "redis", name),
			"secret %q should be accessible via exact-match convention", name)
	}
}

func TestAllow_ExactMatchConvention_NamespaceMismatch_RequiresPolicy(t *testing.T) {
	// SPIFFE ns=payments, sa=api requesting namespace=other — bypass does not apply.
	fs := &fakeStore{} // no policies
	err := allowViaFake(context.Background(), fs,
		"spiffe://cluster.local/ns/payments/sa/api", "get", "other", "api", "secret")
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestAllow_ExactMatchConvention_ServiceMismatch_RequiresPolicy(t *testing.T) {
	// SPIFFE ns=payments, sa=api requesting service=worker — bypass does not apply.
	fs := &fakeStore{}
	err := allowViaFake(context.Background(), fs,
		"spiffe://cluster.local/ns/payments/sa/api", "get", "payments", "worker", "secret")
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestAllow_ExactMatchConvention_NonKubeID_RequiresPolicy(t *testing.T) {
	// A SPIFFE ID that does not follow the /ns/.../sa/... convention never
	// gets the bypass, even if namespace and service happen to match strings
	// embedded in the ID path.
	fs := &fakeStore{}
	err := allowViaFake(context.Background(), fs,
		"spiffe://x/payments/api", "get", "payments", "api", "secret")
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestAllow_PolicyGrant_ViaExplicitPolicy(t *testing.T) {
	// Explicit policies still work for cross-service access patterns.
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://cluster.local/ns/observability/sa/metrics",
		Namespace:   "payments",
		Pattern:     "payments/stripe-key",
		Permissions: []string{"get"},
	}}}
	err := allowViaFake(context.Background(), fs,
		"spiffe://cluster.local/ns/observability/sa/metrics", "get", "payments", "api", "stripe-key")
	require.NoError(t, err)
}

func TestAllow_WildcardPattern(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://x/svc",
		Namespace:   "prod",
		Pattern:     "prod/db-*",
		Permissions: []string{"get"},
	}}}

	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "prod", "svc", "db-password"))
	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "prod", "svc", "db-replica"))
	err := allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "prod", "svc", "redis-url")
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestAllow_StarPermission(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://x/admin",
		Namespace:   "prod",
		Pattern:     "prod/*",
		Permissions: []string{"*"},
	}}}
	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/admin", "get", "prod", "svc", "any-secret"))
	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/admin", "delete", "prod", "svc", "any-secret"))
	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/admin", "list", "prod", "svc", "any-secret"))
}

func TestAllow_WrongPermission(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://x/svc",
		Namespace:   "prod",
		Pattern:     "prod/db-*",
		Permissions: []string{"get"},
	}}}
	err := allowViaFake(context.Background(), fs, "spiffe://x/svc", "delete", "prod", "svc", "db-password")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestAllow_WrongNamespace(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://x/svc",
		Namespace:   "prod",
		Pattern:     "prod/*",
		Permissions: []string{"get"},
	}}}
	err := allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "staging", "svc", "db-password")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestAllow_MultiplePolicies_FirstMatchWins(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{
		{SPIFFEID: "spiffe://x/svc", Namespace: "prod", Pattern: "prod/db-*", Permissions: []string{"get"}},
		{SPIFFEID: "spiffe://x/svc", Namespace: "prod", Pattern: "prod/*", Permissions: []string{"delete"}},
	}}
	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "prod", "svc", "db-password"))
	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/svc", "delete", "prod", "svc", "cache-url"))
}

func TestAllow_DifferentSPIFFE_NoAccess(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://x/svc-a",
		Namespace:   "prod",
		Pattern:     "prod/*",
		Permissions: []string{"get"},
	}}}
	err := allowViaFake(context.Background(), fs, "spiffe://x/svc-b", "get", "prod", "svc", "secret")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestAllow_StoreError_Propagates(t *testing.T) {
	fs := &fakeStore{err: ErrUnauthorized}
	err := allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "prod", "svc", "secret")
	require.Error(t, err)
}

// --- evalPolicies ---

func TestEvalPolicies_EmptyPolicies(t *testing.T) {
	err := evalPolicies(nil, "spiffe://x", "get", "ns", "key")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestEvalPolicies_MatchErrorBadPattern(t *testing.T) {
	// path.Match returns an error on syntactically invalid patterns like "[z-a]".
	// evalPolicies must skip broken patterns without panicking.
	err := evalPolicies([]store.Policy{{
		SPIFFEID:    "s",
		Namespace:   "ns",
		Pattern:     "[z-a]", // invalid range
		Permissions: []string{"get"},
	}}, "s", "get", "ns", "key")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}
