//go:build integration

package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cleanLocks(t *testing.T, s *Store) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), "DELETE FROM restart_locks")
	require.NoError(t, err)
}

func TestTryAcquireLock_FreshAcquire(t *testing.T) {
	s := newTestStore(t)
	cleanLocks(t, s)

	acquired, err := s.TryAcquireLock(context.Background(), "ns", "svc", "token", time.Now().Add(time.Minute))
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestTryAcquireLock_BlockedByLiveHolder(t *testing.T) {
	s := newTestStore(t)
	cleanLocks(t, s)
	ctx := context.Background()

	acquired, err := s.TryAcquireLock(ctx, "ns", "svc", "holder", time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.True(t, acquired)

	acquired, err = s.TryAcquireLock(ctx, "ns", "svc", "challenger", time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, acquired, "a live (non-expired) holder must not be stolen from")
}

// TestTryAcquireLock_StealsExpiredRow verifies the same "steal an expired
// lock" behavior the CockroachDB-specific regression test
// (locks_cockroachdb_integration_test.go) exists to check against real
// CockroachDB — Postgres never rejected the pre-fix query, but the fixed
// query's behavior should be identical on both.
func TestTryAcquireLock_StealsExpiredRow(t *testing.T) {
	s := newTestStore(t)
	cleanLocks(t, s)
	ctx := context.Background()

	_, err := s.pool.Exec(ctx,
		"INSERT INTO restart_locks (namespace, service, token, expires_at) VALUES ($1, $2, $3, $4)",
		"ns", "svc", "stale-token", time.Now().Add(-time.Minute))
	require.NoError(t, err)

	acquired, err := s.TryAcquireLock(ctx, "ns", "svc", "fresh-token", time.Now().Add(time.Minute))
	require.NoError(t, err)
	assert.True(t, acquired, "an expired row must not block a fresh acquisition")
}

func TestHeartbeatLock_ExtendsExpiry(t *testing.T) {
	s := newTestStore(t)
	cleanLocks(t, s)
	ctx := context.Background()

	acquired, err := s.TryAcquireLock(ctx, "ns", "svc", "token", time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.True(t, acquired)

	newExpiry, err := s.HeartbeatLock(ctx, "ns", "svc", "token", time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(time.Hour), newExpiry, 5*time.Second)
}

func TestHeartbeatLock_WrongTokenReturnsNotFound(t *testing.T) {
	s := newTestStore(t)
	cleanLocks(t, s)
	ctx := context.Background()

	acquired, err := s.TryAcquireLock(ctx, "ns", "svc", "token", time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.True(t, acquired)

	_, err = s.HeartbeatLock(ctx, "ns", "svc", "wrong-token", time.Now().Add(time.Hour))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestReleaseLock_RemovesRow(t *testing.T) {
	s := newTestStore(t)
	cleanLocks(t, s)
	ctx := context.Background()

	acquired, err := s.TryAcquireLock(ctx, "ns", "svc", "token", time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.True(t, acquired)

	require.NoError(t, s.ReleaseLock(ctx, "ns", "svc", "token"))

	// Released; a fresh acquisition (different token) must now succeed
	// immediately, without needing to wait for expiry.
	acquired, err = s.TryAcquireLock(ctx, "ns", "svc", "next-token", time.Now().Add(time.Minute))
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestReleaseLock_WrongTokenIsNoop(t *testing.T) {
	s := newTestStore(t)
	cleanLocks(t, s)
	ctx := context.Background()

	acquired, err := s.TryAcquireLock(ctx, "ns", "svc", "token", time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.True(t, acquired)

	require.NoError(t, s.ReleaseLock(ctx, "ns", "svc", "wrong-token"))

	// The real holder's lock must still be in place.
	acquired, err = s.TryAcquireLock(ctx, "ns", "svc", "challenger", time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, acquired)
}

func TestSweepExpiredLocks_RemovesOnlyExpired(t *testing.T) {
	s := newTestStore(t)
	cleanLocks(t, s)
	ctx := context.Background()

	acquired, err := s.TryAcquireLock(ctx, "ns", "expired", "token", time.Now().Add(-time.Minute))
	require.NoError(t, err)
	require.True(t, acquired)

	acquired, err = s.TryAcquireLock(ctx, "ns", "live", "token", time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.True(t, acquired)

	swept, err := s.SweepExpiredLocks(ctx)
	require.NoError(t, err)
	require.Len(t, swept, 1)
	assert.Equal(t, "expired", swept[0].Service)

	// The live lock must survive the sweep.
	acquired, err = s.TryAcquireLock(ctx, "ns", "live", "challenger", time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, acquired)
}
