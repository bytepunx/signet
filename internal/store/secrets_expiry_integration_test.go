//go:build integration

package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetSecret_ExpiredLatestVersionNotServed is the H-5 regression test: a
// secret whose only version has expired must be treated as absent.
func TestGetSecret_ExpiredLatestVersionNotServed(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	past := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "expired-token",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"), ExpiresAt: &past,
	}))

	_, err := s.GetSecret(context.Background(), "ns", "svc", "expired-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetSecret_FutureExpiryStillServed(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	future := time.Now().UTC().Add(time.Hour)
	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "valid-token",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"), ExpiresAt: &future,
	}))

	got, err := s.GetSecret(context.Background(), "ns", "svc", "valid-token")
	require.NoError(t, err)
	assert.Equal(t, "valid-token", got.Name)
}

func TestGetSecret_NoExpiryStillServed(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "no-expiry",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"),
	}))

	got, err := s.GetSecret(context.Background(), "ns", "svc", "no-expiry")
	require.NoError(t, err)
	assert.Nil(t, got.ExpiresAt)
}

// TestGetSecret_ExpiredLatestFallsBackToOlderValidVersion documents the
// chosen semantics: if the newest version has expired but an older version
// of the same secret has not, GetSecret serves the newest non-expired one
// rather than treating the secret as entirely absent.
func TestGetSecret_ExpiredLatestFallsBackToOlderValidVersion(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	future := time.Now().UTC().Add(time.Hour)
	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "rotating-token",
		EncryptedDEK: []byte("dek-v1"), Ciphertext: []byte("ct-v1"), ExpiresAt: &future,
	}))

	past := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "rotating-token",
		EncryptedDEK: []byte("dek-v2"), Ciphertext: []byte("ct-v2"), ExpiresAt: &past,
	}))

	got, err := s.GetSecret(context.Background(), "ns", "svc", "rotating-token")
	require.NoError(t, err)
	assert.Equal(t, 1, got.Version, "must fall back to the older non-expired version")
	assert.Equal(t, []byte("ct-v1"), got.Ciphertext)
}

func TestFetchServiceSecrets_ExpiredSecretOmitted(t *testing.T) {
	s := newTestStore(t)
	cleanSecrets(t, s)

	past := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "expired",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"), ExpiresAt: &past,
	}))
	require.NoError(t, s.PutSecret(context.Background(), &Secret{
		Namespace: "ns", Service: "svc", Name: "valid",
		EncryptedDEK: []byte("dek"), Ciphertext: []byte("ct"),
	}))

	secs, err := s.FetchServiceSecrets(context.Background(), "ns", "svc")
	require.NoError(t, err)
	names := make(map[string]bool)
	for _, sec := range secs {
		names[sec.Name] = true
	}
	assert.True(t, names["valid"])
	assert.False(t, names["expired"], "expired secret must be omitted from the bundle")
}
