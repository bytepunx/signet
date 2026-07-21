//go:build crdb_integration

// This file requires a separate build tag (not "integration") because it
// defines its own TestMain against a real CockroachDB container — Go allows
// only one TestMain per package, and integration_test.go already defines one
// against Postgres. Run with: go test -tags crdb_integration ./internal/store/...
//
// CockroachDB, not Postgres, is what signet actually runs against in every
// real deployment (see deploy/helm/signet's cockroachdb subchart); the
// Postgres-backed suite exists because it's dramatically faster to start and
// catches the vast majority of query bugs, but CockroachDB has real,
// occasionally surprising SQL compatibility gaps that Postgres does not
// share — this file is for the ones that only show up against the real
// thing. See TestTryAcquireLock_CleanupDeletesExpiredRow_CockroachDB's own
// comment for the specific bug that motivated adding this.
package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/cockroachdb"
)

var crdbConnStr string

func TestMain(m *testing.M) {
	ctx := context.Background()

	ctr, err := cockroachdb.Run(ctx, "cockroachdb/cockroach:v24.3.0",
		cockroachdb.WithDatabase("signet_test"),
		cockroachdb.WithInsecure(), // matches how signet's own cockroachdb subchart runs (see deploy/helm/signet)
	)
	if err != nil {
		panic("start cockroachdb container: " + err.Error())
	}
	defer func() {
		_ = ctr.Terminate(ctx) // best effort on cleanup
	}()

	// Not ctr.ConnectionString(ctx): that helper registers a database/sql
	// driver config and returns its lookup key, which pgxpool.New (what
	// store.New actually uses) cannot parse. Build a plain DSN from the
	// container's host/port instead — insecure mode uses "root" with no
	// password, matching signet's own cockroachdb subchart.
	host, err := ctr.Host(ctx)
	if err != nil {
		panic("get host: " + err.Error())
	}
	port, err := ctr.MappedPort(ctx, "26257/tcp")
	if err != nil {
		panic("get mapped port: " + err.Error())
	}
	crdbConnStr = fmt.Sprintf("postgres://root@%s:%s/signet_test?sslmode=disable", host, port.Port())

	m.Run()
}

func newCRDBTestStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	s, err := New(ctx, crdbConnStr)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

// TestTryAcquireLock_StealsExpiredRow_CockroachDB is a regression test for a
// bug found live, during extended smoke testing against a real signet +
// CockroachDB deployment: every restart-lock re-acquisition for a
// (namespace, service) pair that had a previously-expired lock failed.
//
// The original query was a DELETE-then-INSERT expressed as a CTE:
//
//	WITH cleanup AS (DELETE FROM restart_locks WHERE ... RETURNING namespace)
//	INSERT INTO restart_locks (...) VALUES (...) ON CONFLICT (...) DO NOTHING
//	RETURNING token
//
// which CockroachDB rejects outright once the DELETE actually matches a row:
//
//	ERROR: multiple mutations of the same table "restart_locks" are not
//	supported unless they all use INSERT without ON CONFLICT (SQLSTATE 0A000)
//
// Postgres allows this unconditionally, which is why the Postgres-backed
// integration suite never caught it: a fresh test DB never has a
// pre-existing expired lock row for the cleanup DELETE to actually match.
// This test creates that exact precondition — an already-expired lock row —
// before calling TryAcquireLock, which would have failed outright against
// the pre-fix query on this real CockroachDB container. The fix replaces
// the DELETE+INSERT with a single conditional upsert (see TryAcquireLock's
// doc comment) — exactly one mutation of the table, which CockroachDB
// allows.
func TestTryAcquireLock_StealsExpiredRow_CockroachDB(t *testing.T) {
	s := newCRDBTestStore(t)
	ctx := context.Background()

	_, err := s.pool.Exec(ctx, "DELETE FROM restart_locks WHERE namespace = $1 AND service = $2", "ns", "steals-expired")
	require.NoError(t, err)

	// Seed an already-expired lock row directly (bypassing TryAcquireLock,
	// which never creates an expired row itself) so the next TryAcquireLock
	// call has a real expired row to steal.
	_, err = s.pool.Exec(ctx,
		"INSERT INTO restart_locks (namespace, service, token, expires_at) VALUES ($1, $2, $3, $4)",
		"ns", "steals-expired", "stale-token", time.Now().Add(-time.Minute))
	require.NoError(t, err)

	acquired, err := s.TryAcquireLock(ctx, "ns", "steals-expired", "fresh-token", time.Now().Add(time.Minute))
	require.NoError(t, err, "must not fail with the CockroachDB multiple-mutations error")
	assert.True(t, acquired, "an expired row must not block a fresh acquisition")
}

// TestTryAcquireLock_FreshAcquire_CockroachDB verifies the ordinary case (no
// pre-existing row at all) against real CockroachDB.
func TestTryAcquireLock_FreshAcquire_CockroachDB(t *testing.T) {
	s := newCRDBTestStore(t)
	ctx := context.Background()

	_, err := s.pool.Exec(ctx, "DELETE FROM restart_locks WHERE namespace = $1 AND service = $2", "ns", "fresh")
	require.NoError(t, err)

	acquired, err := s.TryAcquireLock(ctx, "ns", "fresh", "token", time.Now().Add(time.Minute))
	require.NoError(t, err)
	assert.True(t, acquired)
}

// TestTryAcquireLock_BlockedByLiveHolder_CockroachDB verifies that a
// not-yet-expired lock correctly blocks a competing acquisition (rather than
// being silently stolen) against real CockroachDB.
func TestTryAcquireLock_BlockedByLiveHolder_CockroachDB(t *testing.T) {
	s := newCRDBTestStore(t)
	ctx := context.Background()

	_, err := s.pool.Exec(ctx, "DELETE FROM restart_locks WHERE namespace = $1 AND service = $2", "ns", "held")
	require.NoError(t, err)

	acquired, err := s.TryAcquireLock(ctx, "ns", "held", "holder-token", time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.True(t, acquired)

	acquired, err = s.TryAcquireLock(ctx, "ns", "held", "challenger-token", time.Now().Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, acquired, "a live (non-expired) holder must not be stolen from")
}
