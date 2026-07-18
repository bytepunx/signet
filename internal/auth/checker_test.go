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
	ListPolicies(ctx context.Context) ([]store.Policy, error)
}

// fakeStore implements policyStore for testing. Unlike the old
// GetPoliciesForSPIFFE, ListPolicies does no filtering at all — matching a
// policy's (possibly glob) spiffe_id against the caller is evalPolicies'
// job, not the store's, so the fake must return everything unfiltered to
// exercise that.
type fakeStore struct {
	policies []store.Policy
	err      error
}

func (f *fakeStore) ListPolicies(_ context.Context) ([]store.Policy, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.policies, nil
}

// testTrustDomain is the fixed trust domain used by allowViaFake, matching
// the "cluster.local" domain used throughout these tests' SPIFFE IDs.
const testTrustDomain = "cluster.local"

// allowViaFake mirrors the full Checker.Allow logic — including the
// exact-match bypass — using a fakeStore so tests don't need a real database.
func allowViaFake(ctx context.Context, fs policyStore, spiffeID, permission, namespace, service, secretName string) error {
	if spiffeID == "" {
		return ErrUnauthenticated
	}
	// Mirror the exact-match bypass from Checker.Allow.
	if spiffeNS, spiffeSA := parseKubeSpiffeID(spiffeID, testTrustDomain); spiffeNS != "" &&
		spiffeNS == namespace && spiffeSA == service {
		return nil
	}
	policies, err := fs.ListPolicies(ctx)
	if err != nil {
		return err
	}
	return evalPolicies(policies, spiffeID, permission, namespace, service, secretName)
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
		Pattern:     "payments/api/stripe-key",
		Permissions: []string{"get"},
	}}}
	err := allowViaFake(context.Background(), fs,
		"spiffe://cluster.local/ns/observability/sa/metrics", "get", "payments", "api", "stripe-key")
	require.NoError(t, err)
}

// TestAllow_PolicyGrant_DistinguishesServiceWithSameSecretName is the H-4
// regression test: two services in the same namespace with a same-named
// secret must be granted independently — a policy scoped to one service must
// not implicitly grant the other.
func TestAllow_PolicyGrant_DistinguishesServiceWithSameSecretName(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://cluster.local/ns/observability/sa/metrics",
		Namespace:   "payments",
		Pattern:     "payments/api/db-password",
		Permissions: []string{"get"},
	}}}
	spiffeID := "spiffe://cluster.local/ns/observability/sa/metrics"

	assert.NoError(t, allowViaFake(context.Background(), fs, spiffeID, "get", "payments", "api", "db-password"),
		"policy scoped to service=api must grant access to api's db-password")

	err := allowViaFake(context.Background(), fs, spiffeID, "get", "payments", "worker", "db-password")
	assert.ErrorIs(t, err, ErrUnauthorized,
		"policy scoped to service=api must NOT grant access to worker's same-named db-password")
}

func TestAllow_WildcardPattern(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://x/svc",
		Namespace:   "prod",
		Pattern:     "prod/svc/db-*",
		Permissions: []string{"get"},
	}}}

	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "prod", "svc", "db-password"))
	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "prod", "svc", "db-replica"))
	err := allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "prod", "svc", "redis-url")
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestAllow_WildcardServiceSegment(t *testing.T) {
	// A policy may wildcard the service segment to grant across every
	// service in a namespace, matching the design's documented pattern shape.
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://x/svc",
		Namespace:   "payments",
		Pattern:     "payments/*/db-read-replica-*",
		Permissions: []string{"get"},
	}}}
	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "payments", "api", "db-read-replica-1"))
	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "payments", "worker", "db-read-replica-2"))
	err := allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "payments", "api", "stripe-api-key")
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestAllow_StarPermission(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://x/admin",
		Namespace:   "prod",
		Pattern:     "prod/*/*",
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
		Pattern:     "prod/svc/db-*",
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
		Pattern:     "prod/*/*",
		Permissions: []string{"get"},
	}}}
	err := allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "staging", "svc", "db-password")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestAllow_MultiplePolicies_FirstMatchWins(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{
		{SPIFFEID: "spiffe://x/svc", Namespace: "prod", Pattern: "prod/svc/db-*", Permissions: []string{"get"}},
		{SPIFFEID: "spiffe://x/svc", Namespace: "prod", Pattern: "prod/*/*", Permissions: []string{"delete"}},
	}}
	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "prod", "svc", "db-password"))
	assert.NoError(t, allowViaFake(context.Background(), fs, "spiffe://x/svc", "delete", "prod", "svc", "cache-url"))
}

func TestAllow_DifferentSPIFFE_NoAccess(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://x/svc-a",
		Namespace:   "prod",
		Pattern:     "prod/*/*",
		Permissions: []string{"get"},
	}}}
	err := allowViaFake(context.Background(), fs, "spiffe://x/svc-b", "get", "prod", "svc", "secret")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

// TestAllow_WildcardSPIFFEIDSegment covers the case this whole matching path
// was built for: granting every workload with a given service account name,
// across every namespace, access to one shared namespace/service — e.g. a
// "smoke-shared/common" secret every "echo" service account should read
// regardless of which namespace it runs in. '*' matches exactly one path
// segment (it does not cross '/'), matching Go's path.Match semantics.
func TestAllow_WildcardSPIFFEIDSegment(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://cluster.local/ns/*/sa/echo",
		Namespace:   "smoke-shared",
		Pattern:     "smoke-shared/common/*",
		Permissions: []string{"get"},
	}}}
	assert.NoError(t, allowViaFake(context.Background(), fs,
		"spiffe://cluster.local/ns/smoke-go/sa/echo", "get", "smoke-shared", "common", "greeting"))
	assert.NoError(t, allowViaFake(context.Background(), fs,
		"spiffe://cluster.local/ns/smoke-python/sa/echo", "get", "smoke-shared", "common", "greeting"))

	// A different service account name in the same namespace shape must not match.
	err := allowViaFake(context.Background(), fs,
		"spiffe://cluster.local/ns/smoke-go/sa/other", "get", "smoke-shared", "common", "greeting")
	assert.ErrorIs(t, err, ErrUnauthorized)
}

// TestAllow_WildcardSPIFFEIDDoesNotCrossSegments documents that '*' in a
// policy's spiffe_id is a single-path-segment wildcard (Go's path.Match), not
// a "**"-style match-everything-after-this-point glob — a pattern ending in
// "/ns/*" should not also match "/ns/a/b".
func TestAllow_WildcardSPIFFEIDDoesNotCrossSegments(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://cluster.local/ns/*",
		Namespace:   "prod",
		Pattern:     "prod/*/*",
		Permissions: []string{"get"},
	}}}
	err := allowViaFake(context.Background(), fs,
		"spiffe://cluster.local/ns/a/sa/b", "get", "prod", "svc", "secret")
	assert.ErrorIs(t, err, ErrUnauthorized)
}

// TestAllow_WildcardSPIFFEIDEveryWorkload covers the trust-domain-wide grant
// documented in docs/policies.md — since SPIFFE IDs here are always the
// fixed ns/<namespace>/sa/<service> shape, "any workload" is expressed as two
// single-segment wildcards, not a bare "**" (which path.Match doesn't
// support as a cross-segment wildcard at all).
func TestAllow_WildcardSPIFFEIDEveryWorkload(t *testing.T) {
	fs := &fakeStore{policies: []store.Policy{{
		SPIFFEID:    "spiffe://cluster.local/ns/*/sa/*",
		Namespace:   "shared",
		Pattern:     "shared/*/*",
		Permissions: []string{"get"},
	}}}
	assert.NoError(t, allowViaFake(context.Background(), fs,
		"spiffe://cluster.local/ns/anything/sa/anything-else", "get", "shared", "svc", "secret"))
}

func TestAllow_StoreError_Propagates(t *testing.T) {
	fs := &fakeStore{err: ErrUnauthorized}
	err := allowViaFake(context.Background(), fs, "spiffe://x/svc", "get", "prod", "svc", "secret")
	require.Error(t, err)
}

// --- evalPolicies ---

func TestEvalPolicies_EmptyPolicies(t *testing.T) {
	err := evalPolicies(nil, "spiffe://x", "get", "ns", "svc", "key")
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
	}}, "s", "get", "ns", "svc", "key")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}
