package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// LockKey identifies a restart lock by its namespace and service.
type LockKey struct {
	Namespace string
	Service   string
}

// TryAcquireLock attempts to insert a new lock record for (namespace, service).
// Any existing record whose expires_at is in the past is removed first.
// Returns true if the lock was acquired, false if currently held by another holder.
// The caller supplies both the token (unique per acquisition) and the expiry time.
func (s *Store) TryAcquireLock(ctx context.Context, namespace, service, token string, expiresAt time.Time) (bool, error) {
	const q = `
		WITH cleanup AS (
			DELETE FROM restart_locks
			WHERE namespace = $1 AND service = $2 AND expires_at < now()
		)
		INSERT INTO restart_locks (namespace, service, token, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (namespace, service) DO NOTHING
		RETURNING token`

	var got string
	err := s.pool.QueryRow(ctx, q, namespace, service, token, expiresAt).Scan(&got)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, wrapDBError("try acquire lock", err)
	}
	return true, nil
}

// HeartbeatLock updates the expiry of a held lock.
// Returns the new expiry, or ErrNotFound if the token no longer matches
// (the lock expired and was re-acquired by a different holder).
func (s *Store) HeartbeatLock(ctx context.Context, namespace, service, token string, expiresAt time.Time) (time.Time, error) {
	const q = `
		UPDATE restart_locks
		SET expires_at = $4
		WHERE namespace = $1 AND service = $2 AND token = $3
		RETURNING expires_at`

	var newExpiry time.Time
	err := s.pool.QueryRow(ctx, q, namespace, service, token, expiresAt).Scan(&newExpiry)
	if err != nil {
		if err == pgx.ErrNoRows {
			return time.Time{}, ErrNotFound
		}
		return time.Time{}, wrapDBError("heartbeat lock", err)
	}
	return newExpiry, nil
}

// ReleaseLock deletes the lock record for (namespace, service) only if the
// token matches the current holder. A mismatch (expired + re-acquired) is
// treated as a no-op rather than an error — the new holder retains the lock.
func (s *Store) ReleaseLock(ctx context.Context, namespace, service, token string) error {
	const q = `
		DELETE FROM restart_locks
		WHERE namespace = $1 AND service = $2 AND token = $3`

	if _, err := s.pool.Exec(ctx, q, namespace, service, token); err != nil {
		return wrapDBError("release lock", err)
	}
	return nil
}

// SweepExpiredLocks deletes all lock records whose expires_at has passed and
// returns the (namespace, service) pairs that were removed. The lock manager
// calls this on a background ticker to unblock waiting processes whose holder
// died without releasing or heartbeating.
func (s *Store) SweepExpiredLocks(ctx context.Context) ([]LockKey, error) {
	const q = `
		DELETE FROM restart_locks
		WHERE expires_at < now()
		RETURNING namespace, service`

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("sweep expired locks: %w", err)
	}
	defer rows.Close()

	var keys []LockKey
	for rows.Next() {
		var k LockKey
		if err := rows.Scan(&k.Namespace, &k.Service); err != nil {
			return nil, fmt.Errorf("sweep expired locks: scan: %w", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sweep expired locks: %w", err)
	}
	return keys, nil
}
