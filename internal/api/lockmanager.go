package api

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bytepunx/signet/internal/store"
)

// lockWaiter represents a single process waiting to acquire a restart lock.
// Both channels are owned by the LockManager; the handler selects on them.
type lockWaiter struct {
	// notify receives a struct{} when the waiter should attempt DB acquisition.
	// Buffered (1) so NotifyFirst never blocks.
	notify chan struct{}
	// posUpdates receives the caller's new queue position whenever it changes.
	// Buffered (4) to absorb bursts when many waiters shift at once.
	posUpdates chan int32
}

// LockManager coordinates restart lock acquisition across clients connected to
// this signet instance. The DB is the authoritative source of truth; this struct
// holds in-memory wait queues for fast same-instance notification.
//
// Multiple signet replicas use the DB for cross-instance coordination: a
// background sweeper removes expired locks and notifies local waiters, who then
// race to re-acquire via DB. Only one wins per DB transaction.
type LockManager struct {
	store lockStore

	mu     sync.Mutex
	queues map[string][]*lockWaiter // key = lockKey(namespace, service)
}

// NewLockManager returns a LockManager backed by the given store.
// Call Run to start the background TTL sweeper.
func NewLockManager(s lockStore) *LockManager {
	return &LockManager{
		store:  s,
		queues: make(map[string][]*lockWaiter),
	}
}

// Run starts the background sweeper that removes expired lock records and
// notifies local waiters. It blocks until ctx is cancelled.
func (m *LockManager) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			swept, err := m.store.SweepExpiredLocks(ctx)
			if err != nil {
				slog.Error("lock sweeper: sweep expired locks", "err", err)
				continue
			}
			for _, key := range swept {
				slog.Info("lock sweeper: expired lock removed", "namespace", key.Namespace, "service", key.Service)
				m.NotifyFirst(key.Namespace, key.Service)
			}
		}
	}
}

// TryAcquire attempts to atomically acquire the lock in the DB using the supplied
// token. Returns (token, expiresAt, true, nil) on success, or ("", zero, false, nil)
// if the lock is currently held.
func (m *LockManager) TryAcquire(ctx context.Context, ns, svc, token string, ttl time.Duration) (time.Time, bool, error) {
	expiresAt := time.Now().Add(ttl)
	ok, err := m.store.TryAcquireLock(ctx, ns, svc, token, expiresAt)
	if err != nil {
		return time.Time{}, false, err
	}
	return expiresAt, ok, nil
}

// Heartbeat extends the lock expiry. Returns the new expiry, or store.ErrNotFound
// if the token no longer matches (the lock expired and was re-acquired).
func (m *LockManager) Heartbeat(ctx context.Context, ns, svc, token string, ttl time.Duration) (time.Time, error) {
	return m.store.HeartbeatLock(ctx, ns, svc, token, time.Now().Add(ttl))
}

// Release deletes the lock record (token-guarded) and notifies the first waiter
// in the local queue to attempt acquisition. Errors are logged but not returned —
// the lock will expire via TTL sweep if the delete fails.
func (m *LockManager) Release(ctx context.Context, ns, svc, token string) {
	if err := m.store.ReleaseLock(ctx, ns, svc, token); err != nil {
		slog.Warn("lock manager: release lock", "namespace", ns, "service", svc, "err", err)
	}
	m.NotifyFirst(ns, svc)
}

// Enqueue adds a new waiter to the back of the queue for (namespace, service)
// and returns the waiter and its initial 1-based queue position.
func (m *LockManager) Enqueue(ns, svc string) (*lockWaiter, int32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w := &lockWaiter{
		notify:     make(chan struct{}, 1),
		posUpdates: make(chan int32, 4),
	}
	key := lockManagerKey(ns, svc)
	m.queues[key] = append(m.queues[key], w)
	return w, int32(len(m.queues[key]))
}

// Dequeue removes a waiter from the queue. Remaining waiters that shifted
// forward receive their new position on their posUpdates channel.
// Idempotent: safe to call if the waiter has already been removed.
func (m *LockManager) Dequeue(ns, svc string, w *lockWaiter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := lockManagerKey(ns, svc)
	queue := m.queues[key]
	for i, candidate := range queue {
		if candidate != w {
			continue
		}
		m.queues[key] = append(queue[:i], queue[i+1:]...)
		if len(m.queues[key]) == 0 {
			delete(m.queues, key)
		}
		// Notify any waiters that shifted forward.
		for j, shifted := range m.queues[key][i:] {
			newPos := int32(i + j + 1)
			select {
			case shifted.posUpdates <- newPos:
			default:
			}
		}
		return
	}
}

// NotifyFirst signals the first waiter in the queue (if any) to attempt
// DB acquisition. Called after a lock is released or swept.
func (m *LockManager) NotifyFirst(ns, svc string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := lockManagerKey(ns, svc)
	if len(m.queues[key]) == 0 {
		return
	}
	first := m.queues[key][0]
	select {
	case first.notify <- struct{}{}:
	default:
	}
}

func lockManagerKey(namespace, service string) string {
	return namespace + "\x00" + service
}

// lockStore is the subset of *store.Store used by LockManager.
type lockStore interface {
	TryAcquireLock(ctx context.Context, namespace, service, token string, expiresAt time.Time) (bool, error)
	HeartbeatLock(ctx context.Context, namespace, service, token string, expiresAt time.Time) (time.Time, error)
	ReleaseLock(ctx context.Context, namespace, service, token string) error
	SweepExpiredLocks(ctx context.Context) ([]store.LockKey, error)
}
