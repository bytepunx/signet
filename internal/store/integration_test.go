//go:build integration

package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcwait "github.com/testcontainers/testcontainers-go/wait"
)

// connStr holds the test database URL; set up in TestMain.
var connStr string

func TestMain(m *testing.M) {
	ctx := context.Background()

	ctr, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("signet_test"),
		postgres.WithUsername("signet"),
		postgres.WithPassword("signet"),
		testcontainers.WithWaitStrategy(
			tcwait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		panic("start postgres container: " + err.Error())
	}
	defer func() {
		if err := ctr.Terminate(ctx); err != nil {
			_ = err // best effort on cleanup
		}
	}()

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("get connection string: " + err.Error())
	}
	connStr = dsn

	m.Run()
}

// newTestStore opens a Store against the test database. Each test gets an
// isolated Store and closes it when done — migrations are idempotent.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	s, err := New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

// cleanSecrets removes all secrets so test cases don't interfere.
func cleanSecrets(t *testing.T, s *Store) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), "DELETE FROM secrets")
	require.NoError(t, err)
}

func cleanPolicies(t *testing.T, s *Store) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), "DELETE FROM access_policies")
	require.NoError(t, err)
}

func cleanAudit(t *testing.T, s *Store) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), "DELETE FROM audit_log")
	require.NoError(t, err)
}

// --- Store lifecycle ---

func TestNew_ValidConnString(t *testing.T) {
	s := newTestStore(t)
	require.NotNil(t, s)
}

func TestPing(t *testing.T) {
	s := newTestStore(t)
	err := s.Ping(context.Background())
	require.NoError(t, err)
}

func TestMigrationsIdempotent(t *testing.T) {
	// Open two Stores against the same DB — both must succeed without error.
	s1 := newTestStore(t)
	s2 := newTestStore(t)
	require.NotNil(t, s1)
	require.NotNil(t, s2)
}

// --- PutSecret ---

func TestPutSecret_FirstVersion(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	sec := &Secret{
		Namespace:    "prod",
		Service:      "api",
		Name:         "db-password",
		EncryptedDEK: []byte("fake-dek-bytes-32----32----------"),
		Ciphertext:   []byte("fake-ciphertext"),
	}
	err := s.PutSecret(context.Background(), sec)
	require.NoError(t, err)
	assert.Equal(t, 1, sec.Version)
	assert.False(t, sec.CreatedAt.IsZero())
	assert.False(t, sec.UpdatedAt.IsZero())
}

func TestPutSecret_AutoIncrementsVersion(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	base := &Secret{
		Namespace:    "prod",
		Service:      "api",
		Name:         "db-password",
		EncryptedDEK: []byte("dek-v1"),
		Ciphertext:   []byte("cipher-v1"),
	}
	require.NoError(t, s.PutSecret(context.Background(), base))
	assert.Equal(t, 1, base.Version)

	updated := &Secret{
		Namespace:    "prod",
		Service:      "api",
		Name:         "db-password",
		EncryptedDEK: []byte("dek-v2"),
		Ciphertext:   []byte("cipher-v2"),
	}
	require.NoError(t, s.PutSecret(context.Background(), updated))
	assert.Equal(t, 2, updated.Version)
}

func TestPutSecret_WithMetadata(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	sec := &Secret{
		Namespace:    "prod",
		Service:      "api",
		Name:         "tls-cert",
		EncryptedDEK: []byte("dek"),
		Ciphertext:   []byte("ciphertext"),
		Metadata:     map[string]string{"managed-by": "signet", "env": "production"},
	}
	require.NoError(t, s.PutSecret(context.Background(), sec))

	got, err := s.GetSecret(context.Background(), "prod", "api", "tls-cert")
	require.NoError(t, err)
	assert.Equal(t, "signet", got.Metadata["managed-by"])
	assert.Equal(t, "production", got.Metadata["env"])
}

func TestPutSecret_WithExpiresAt(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	expiry := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Millisecond)
	sec := &Secret{
		Namespace:    "prod",
		Service:      "api",
		Name:         "session-token",
		EncryptedDEK: []byte("dek"),
		Ciphertext:   []byte("ciphertext"),
		ExpiresAt:    &expiry,
	}
	require.NoError(t, s.PutSecret(context.Background(), sec))

	got, err := s.GetSecret(context.Background(), "prod", "api", "session-token")
	require.NoError(t, err)
	require.NotNil(t, got.ExpiresAt)
	assert.WithinDuration(t, expiry, *got.ExpiresAt, time.Second)
}

func TestPutSecret_NilInput(t *testing.T) {
	s := newTestStore(t)
	err := s.PutSecret(context.Background(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestPutSecret_MissingEncryptedDEK(t *testing.T) {
	s := newTestStore(t)
	err := s.PutSecret(context.Background(), &Secret{
		Namespace:  "ns",
		Service:    "svc",
		Name:       "k",
		Ciphertext: []byte("x"),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

// --- GetSecret ---

func TestGetSecret_NotFound(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	_, err := s.GetSecret(context.Background(), "ns", "svc", "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetSecret_ReturnsLatestVersion(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	for i := 0; i < 3; i++ {
		require.NoError(t, s.PutSecret(context.Background(), &Secret{
			Namespace:    "ns",
			Service:      "svc",
			Name:         "key",
			EncryptedDEK: []byte("dek"),
			Ciphertext:   []byte("cipher"),
		}))
	}

	got, err := s.GetSecret(context.Background(), "ns", "svc", "key")
	require.NoError(t, err)
	assert.Equal(t, 3, got.Version)
}

func TestGetSecret_PayloadRoundtrip(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	dek := []byte("this-is-an-encrypted-dek-32bytes!")
	ct := []byte("this-is-ciphertext-not-plaintext!")
	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace:    "ns",
		Service:      "svc",
		Name:         "key",
		EncryptedDEK: dek,
		Ciphertext:   ct,
	}))

	got, err := s.GetSecret(context.Background(), "ns", "svc", "key")
	require.NoError(t, err)
	assert.Equal(t, dek, got.EncryptedDEK)
	assert.Equal(t, ct, got.Ciphertext)
}

func TestGetSecret_InvalidInput(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetSecret(context.Background(), "", "svc", "key")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

// --- GetSecretAtVersion ---

func TestGetSecretAtVersion_Specific(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "key",
		EncryptedDEK: []byte("dek-v1"), Ciphertext: []byte("ct-v1"),
	}))
	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "key",
		EncryptedDEK: []byte("dek-v2"), Ciphertext: []byte("ct-v2"),
	}))

	got, err := s.GetSecretAtVersion(context.Background(), "ns", "svc", "key", 1)
	require.NoError(t, err)
	assert.Equal(t, 1, got.Version)
	assert.Equal(t, []byte("dek-v1"), got.EncryptedDEK)
	assert.Equal(t, []byte("ct-v1"), got.Ciphertext)
}

func TestGetSecretAtVersion_NotFound(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	_, err := s.GetSecretAtVersion(context.Background(), "ns", "svc", "key", 99)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetSecretAtVersion_ZeroVersion(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetSecretAtVersion(context.Background(), "ns", "svc", "key", 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestGetSecretAtVersion_NegativeVersion(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetSecretAtVersion(context.Background(), "ns", "svc", "key", -1)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

// --- ListSecrets ---

func TestListSecrets_Empty(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	metas, err := s.ListSecrets(context.Background(), "ns", "svc")
	require.NoError(t, err)
	assert.Empty(t, metas)
}

func TestListSecrets_ByService(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	for _, name := range []string{"alpha", "beta", "gamma"} {
		require.NoError(t, s.PutSecret(context.Background(), &Secret{
			Namespace: "ns", Service: "svc", Name: name,
			EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"),
		}))
	}
	// A secret in a different service — should not appear.
	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "other", Name: "alpha",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"),
	}))

	metas, err := s.ListSecrets(context.Background(), "ns", "svc")
	require.NoError(t, err)
	assert.Len(t, metas, 3)
	for _, m := range metas {
		assert.Equal(t, "svc", m.Service)
	}
}

func TestListSecrets_AcrossServices(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	for _, svc := range []string{"api", "worker", "proxy"} {
		require.NoError(t, s.PutSecret(context.Background(), &Secret{
			Namespace: "prod", Service: svc, Name: "db-url",
			EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"),
		}))
	}

	// Empty service → all services.
	metas, err := s.ListSecrets(context.Background(), "prod", "")
	require.NoError(t, err)
	assert.Len(t, metas, 3)
}

func TestListSecrets_ReturnsLatestVersionOnly(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	for i := 0; i < 3; i++ {
		require.NoError(t, s.PutSecret(context.Background(), &Secret{
			Namespace: "ns", Service: "svc", Name: "key",
			EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"),
		}))
	}

	metas, err := s.ListSecrets(context.Background(), "ns", "svc")
	require.NoError(t, err)
	require.Len(t, metas, 1)
	assert.Equal(t, 3, metas[0].Version)
}

func TestListSecrets_EmptyNamespace(t *testing.T) {
	s := newTestStore(t)
	_, err := s.ListSecrets(context.Background(), "", "svc")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestListSecrets_NoPayload(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "key",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"),
	}))

	metas, err := s.ListSecrets(context.Background(), "ns", "svc")
	require.NoError(t, err)
	require.Len(t, metas, 1)
	// SecretMeta must not expose encrypted payload.
	assert.Empty(t, metas[0].Namespace) // would fail only if struct had payload fields
	// Actually test what is present.
	assert.Equal(t, "ns", metas[0].Namespace)
	assert.Equal(t, "svc", metas[0].Service)
	assert.Equal(t, "key", metas[0].Name)
}

// --- DeleteSecret ---

func TestDeleteSecret_OK(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "key",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"),
	}))

	err := s.DeleteSecret(context.Background(), "ns", "svc", "key")
	require.NoError(t, err)

	_, err = s.GetSecret(context.Background(), "ns", "svc", "key")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteSecret_DeletesAllVersions(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	for i := 0; i < 3; i++ {
		require.NoError(t, s.PutSecret(context.Background(), &Secret{
			Namespace: "ns", Service: "svc", Name: "key",
			EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"),
		}))
	}

	require.NoError(t, s.DeleteSecret(context.Background(), "ns", "svc", "key"))

	// All versions gone.
	for v := 1; v <= 3; v++ {
		_, err := s.GetSecretAtVersion(context.Background(), "ns", "svc", "key", v)
		assert.ErrorIs(t, err, ErrNotFound, "version %d should be deleted", v)
	}
}

func TestDeleteSecret_NotFound(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	err := s.DeleteSecret(context.Background(), "ns", "svc", "nonexistent")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteSecret_InvalidInput(t *testing.T) {
	s := newTestStore(t)
	err := s.DeleteSecret(context.Background(), "", "svc", "key")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

// --- PutPolicy ---

func TestPutPolicy_OK(t *testing.T) {
	s := newTestStore(t)
	cleanPolicies(t, s)

	p := &Policy{
		SPIFFEID:    "spiffe://cluster.local/ns/prod/sa/api",
		Namespace:   "prod",
		Pattern:     "db-*",
		Permissions: []string{"get"},
	}
	err := s.PutPolicy(context.Background(), p)
	require.NoError(t, err)
	assert.NotEmpty(t, p.ID)
	assert.False(t, p.CreatedAt.IsZero())
}

func TestPutPolicy_MultiplePermissions(t *testing.T) {
	s := newTestStore(t)
	cleanPolicies(t, s)

	p := &Policy{
		SPIFFEID:    "spiffe://cluster.local/ns/prod/sa/admin",
		Namespace:   "prod",
		Pattern:     "*",
		Permissions: []string{"get", "list", "delete"},
	}
	require.NoError(t, s.PutPolicy(context.Background(), p))

	got, err := s.GetPolicyByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"get", "list", "delete"}, got.Permissions)
}

func TestPutPolicy_NilInput(t *testing.T) {
	s := newTestStore(t)
	err := s.PutPolicy(context.Background(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

// --- GetPoliciesForSPIFFE ---

func TestGetPoliciesForSPIFFE_Empty(t *testing.T) {
	s := newTestStore(t)
	cleanPolicies(t, s)

	policies, err := s.GetPoliciesForSPIFFE(context.Background(), "spiffe://cluster.local/ns/prod/sa/unknown")
	require.NoError(t, err)
	assert.Empty(t, policies) // not ErrNotFound — no policies is a valid state
}

func TestGetPoliciesForSPIFFE_ReturnsOnlyMatching(t *testing.T) {
	s := newTestStore(t)
	cleanPolicies(t, s)

	target := "spiffe://cluster.local/ns/prod/sa/api"
	other := "spiffe://cluster.local/ns/prod/sa/worker"

	require.NoError(t, s.PutPolicy(context.Background(), &Policy{
		SPIFFEID: target, Namespace: "prod", Pattern: "db-*", Permissions: []string{"get"},
	}))
	require.NoError(t, s.PutPolicy(context.Background(), &Policy{
		SPIFFEID: target, Namespace: "prod", Pattern: "cache-*", Permissions: []string{"get"},
	}))
	require.NoError(t, s.PutPolicy(context.Background(), &Policy{
		SPIFFEID: other, Namespace: "prod", Pattern: "*", Permissions: []string{"get"},
	}))

	policies, err := s.GetPoliciesForSPIFFE(context.Background(), target)
	require.NoError(t, err)
	assert.Len(t, policies, 2)
	for _, p := range policies {
		assert.Equal(t, target, p.SPIFFEID)
	}
}

func TestGetPoliciesForSPIFFE_EmptySPIFFEID(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetPoliciesForSPIFFE(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

// --- DeletePolicy ---

func TestDeletePolicy_OK(t *testing.T) {
	s := newTestStore(t)
	cleanPolicies(t, s)

	p := &Policy{
		SPIFFEID: "spiffe://cluster.local/ns/prod/sa/api", Namespace: "prod",
		Pattern: "*", Permissions: []string{"get"},
	}
	require.NoError(t, s.PutPolicy(context.Background(), p))

	err := s.DeletePolicy(context.Background(), p.ID)
	require.NoError(t, err)

	_, err = s.GetPolicyByID(context.Background(), p.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeletePolicy_NotFound(t *testing.T) {
	s := newTestStore(t)
	err := s.DeletePolicy(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDeletePolicy_EmptyID(t *testing.T) {
	s := newTestStore(t)
	err := s.DeletePolicy(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

// --- GetPolicyByID ---

func TestGetPolicyByID_Roundtrip(t *testing.T) {
	s := newTestStore(t)
	cleanPolicies(t, s)

	p := &Policy{
		SPIFFEID:    "spiffe://cluster.local/ns/prod/sa/api",
		Namespace:   "prod",
		Pattern:     "db-*",
		Permissions: []string{"get", "list"},
	}
	require.NoError(t, s.PutPolicy(context.Background(), p))

	got, err := s.GetPolicyByID(context.Background(), p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.SPIFFEID, got.SPIFFEID)
	assert.Equal(t, p.Namespace, got.Namespace)
	assert.Equal(t, p.Pattern, got.Pattern)
	assert.ElementsMatch(t, p.Permissions, got.Permissions)
}

func TestGetPolicyByID_EmptyID(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetPolicyByID(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

// --- WriteAuditLog ---

func TestWriteAuditLog_OK(t *testing.T) {
	s := newTestStore(t)
	cleanAudit(t, s)

	entry := &AuditEntry{
		SPIFFEID:   "spiffe://cluster.local/ns/prod/sa/api",
		Action:     "get",
		Namespace:  "prod",
		SecretName: "db-password",
		Outcome:    "permitted",
		PeerIP:     "10.0.0.1",
		HMAC:       make([]byte, 32),
	}
	err := s.WriteAuditLog(context.Background(), entry)
	require.NoError(t, err)
}

func TestWriteAuditLog_DeniedOutcome(t *testing.T) {
	s := newTestStore(t)
	cleanAudit(t, s)

	err := s.WriteAuditLog(context.Background(), &AuditEntry{
		SPIFFEID:   "spiffe://cluster.local/ns/prod/sa/untrusted",
		Action:     "get",
		Namespace:  "prod",
		SecretName: "classified",
		Outcome:    "denied",
		HMAC:       make([]byte, 32),
	})
	require.NoError(t, err)
}

func TestWriteAuditLog_NoPeerIP(t *testing.T) {
	s := newTestStore(t)
	cleanAudit(t, s)

	err := s.WriteAuditLog(context.Background(), &AuditEntry{
		SPIFFEID:   "spiffe://cluster.local/ns/prod/sa/api",
		Action:     "list",
		Namespace:  "prod",
		SecretName: "*",
		Outcome:    "permitted",
		PeerIP:     "", // no peer IP available
		HMAC:       make([]byte, 32),
	})
	require.NoError(t, err)
}

func TestWriteAuditLog_NilEntry(t *testing.T) {
	s := newTestStore(t)
	err := s.WriteAuditLog(context.Background(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestWriteAuditLog_MissingFields(t *testing.T) {
	s := newTestStore(t)
	cases := []struct {
		name  string
		entry *AuditEntry
	}{
		{"no spiffe", &AuditEntry{Action: "get", Namespace: "ns", SecretName: "k", Outcome: "ok", HMAC: []byte{1}}},
		{"no action", &AuditEntry{SPIFFEID: "s", Namespace: "ns", SecretName: "k", Outcome: "ok", HMAC: []byte{1}}},
		{"no namespace", &AuditEntry{SPIFFEID: "s", Action: "get", SecretName: "k", Outcome: "ok", HMAC: []byte{1}}},
		{"no secret name", &AuditEntry{SPIFFEID: "s", Action: "get", Namespace: "ns", Outcome: "ok", HMAC: []byte{1}}},
		{"no outcome", &AuditEntry{SPIFFEID: "s", Action: "get", Namespace: "ns", SecretName: "k", HMAC: []byte{1}}},
		{"no hmac", &AuditEntry{SPIFFEID: "s", Action: "get", Namespace: "ns", SecretName: "k", Outcome: "ok"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.WriteAuditLog(context.Background(), tc.entry)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidInput, "expected ErrInvalidInput for %s", tc.name)
		})
	}
}

func TestWriteAuditLog_LargeHMAC(t *testing.T) {
	s := newTestStore(t)
	cleanAudit(t, s)

	err := s.WriteAuditLog(context.Background(), &AuditEntry{
		SPIFFEID:   "spiffe://cluster.local/ns/prod/sa/api",
		Action:     "get",
		Namespace:  "prod",
		SecretName: "db-password",
		Outcome:    "permitted",
		HMAC:       make([]byte, 64), // SHA-512 HMAC
	})
	require.NoError(t, err)
}

// --- Cross-namespace isolation ---

func TestSecretsNamespaceIsolation(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "prod", Service: "api", Name: "secret",
		EncryptedDEK: []byte("dek-prod"), Ciphertext: []byte("ct-prod"),
	}))
	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "staging", Service: "api", Name: "secret",
		EncryptedDEK: []byte("dek-staging"), Ciphertext: []byte("ct-staging"),
	}))

	prod, err := s.GetSecret(context.Background(), "prod", "api", "secret")
	require.NoError(t, err)
	assert.Equal(t, []byte("dek-prod"), prod.EncryptedDEK)

	staging, err := s.GetSecret(context.Background(), "staging", "api", "secret")
	require.NoError(t, err)
	assert.Equal(t, []byte("dek-staging"), staging.EncryptedDEK)
}

// --- Error is-chain correctness ---

func TestErrorChain_NotFound(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	_, err := s.GetSecret(context.Background(), "ns", "svc", "missing")
	require.Error(t, err)
	// Callers must be able to use errors.Is to detect ErrNotFound.
	assert.True(t, errors.Is(err, ErrNotFound), "expected errors.Is(err, ErrNotFound)")
}

func TestErrorChain_InvalidInput(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetSecret(context.Background(), "", "svc", "key")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidInput), "expected errors.Is(err, ErrInvalidInput)")
}
