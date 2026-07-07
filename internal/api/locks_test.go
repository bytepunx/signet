package api

import (
	"context"
	"sync"
	"testing"
	"time"

	signetv1 "github.com/bytepunx/signet/gen/signet/v1"
	"github.com/bytepunx/signet/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// --- controllable lockStore for lock tests ---

type controlLockStore struct {
	mu      sync.Mutex
	held    map[string]string // key → token of current holder
	results []bool            // controlled TryAcquire results; cycles if exhausted
}

func newControlLockStore() *controlLockStore {
	return &controlLockStore{held: make(map[string]string)}
}

func (s *controlLockStore) TryAcquireLock(_ context.Context, ns, svc, token string, _ time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ns + "\x00" + svc
	if _, held := s.held[key]; held {
		return false, nil
	}
	s.held[key] = token
	return true, nil
}

func (s *controlLockStore) HeartbeatLock(_ context.Context, ns, svc, token string, _ time.Time) (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ns + "\x00" + svc
	if s.held[key] != token {
		return time.Time{}, store.ErrNotFound
	}
	return time.Now().Add(60 * time.Second), nil
}

func (s *controlLockStore) ReleaseLock(_ context.Context, ns, svc, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ns + "\x00" + svc
	if s.held[key] == token {
		delete(s.held, key)
	}
	return nil
}

func (s *controlLockStore) SweepExpiredLocks(_ context.Context) ([]store.LockKey, error) {
	return nil, nil
}

// forceRelease removes the lock unconditionally for test sweep simulation.
func (s *controlLockStore) forceRelease(ns, svc string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.held, ns+"\x00"+svc)
}

// --- fake bidirectional stream ---

type fakeLockStream struct {
	ctx     context.Context
	recvCh  chan *signetv1.AcquireRestartLockRequest
	sendCh  chan *signetv1.AcquireRestartLockResponse
	sendErr error
}

func newFakeLockStream(ctx context.Context) *fakeLockStream {
	return &fakeLockStream{
		ctx:    ctx,
		recvCh: make(chan *signetv1.AcquireRestartLockRequest, 16),
		sendCh: make(chan *signetv1.AcquireRestartLockResponse, 16),
	}
}

func (f *fakeLockStream) Send(r *signetv1.AcquireRestartLockResponse) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sendCh <- r
	return nil
}

func (f *fakeLockStream) Recv() (*signetv1.AcquireRestartLockRequest, error) {
	select {
	case <-f.ctx.Done():
		return nil, f.ctx.Err()
	case msg := <-f.recvCh:
		return msg, nil
	}
}

func (f *fakeLockStream) Context() context.Context     { return f.ctx }
func (f *fakeLockStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeLockStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeLockStream) SetTrailer(metadata.MD)       {}
func (f *fakeLockStream) SendMsg(any) error            { return nil }
func (f *fakeLockStream) RecvMsg(any) error            { return nil }

// sendInit enqueues an initial lock request on the stream's receive channel.
func (f *fakeLockStream) sendInit(ns, svc string, ttlSecs int32) {
	f.recvCh <- &signetv1.AcquireRestartLockRequest{
		Namespace:  ns,
		Service:    svc,
		TtlSeconds: ttlSecs,
	}
}

func (f *fakeLockStream) sendHeartbeat(ttlSecs int32) {
	f.recvCh <- &signetv1.AcquireRestartLockRequest{Heartbeat: true, TtlSeconds: ttlSecs}
}

// --- helpers ---

func newLockServer() (*SecretsServer, *controlLockStore, *LockManager) {
	ls := newControlLockStore()
	lm := NewLockManager(ls)
	srv := NewSecretsServer(&fakeSecretFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), lm, false)
	return srv, ls, lm
}

// --- tests ---

func TestAcquireRestartLock_MissingNamespace(t *testing.T) {
	srv, _, _ := newLockServer()
	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	defer cancel()
	stream := newFakeLockStream(ctx)
	stream.sendInit("", "svc", 30)
	err := srv.AcquireRestartLock(stream)
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestAcquireRestartLock_MissingTTL(t *testing.T) {
	srv, _, _ := newLockServer()
	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	defer cancel()
	stream := newFakeLockStream(ctx)
	stream.sendInit("ns", "svc", 0)
	err := srv.AcquireRestartLock(stream)
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument for zero ttl, got %v", err)
	}
}

func TestAcquireRestartLock_NoMTLS_Unauthenticated(t *testing.T) {
	srv, _, _ := newLockServer()
	ctx, cancel := context.WithCancel(context.Background()) // no peer
	defer cancel()
	stream := newFakeLockStream(ctx)
	stream.sendInit("ns", "svc", 30)
	err := srv.AcquireRestartLock(stream)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("want Unauthenticated, got %v", err)
	}
}

func TestAcquireRestartLock_ImmediateAcquire(t *testing.T) {
	srv, _, _ := newLockServer()
	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	stream := newFakeLockStream(ctx)
	stream.sendInit("ns", "svc", 30)

	done := make(chan error, 1)
	go func() { done <- srv.AcquireRestartLock(stream) }()

	// Should receive ACQUIRED immediately (lock was free).
	select {
	case msg := <-stream.sendCh:
		if msg.MessageType != signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_ACQUIRED {
			t.Errorf("want ACQUIRED, got %v", msg.MessageType)
		}
		if msg.Token == "" {
			t.Error("token must not be empty")
		}
		if msg.ExpiresAt == nil {
			t.Error("expires_at must be set")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ACQUIRED")
	}

	cancel()
	<-done
}

func TestAcquireRestartLock_HeartbeatExtendsExpiry(t *testing.T) {
	srv, _, _ := newLockServer()
	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	stream := newFakeLockStream(ctx)
	stream.sendInit("ns", "svc", 30)

	go srv.AcquireRestartLock(stream) //nolint:errcheck

	// Wait for ACQUIRED.
	var acquired *signetv1.AcquireRestartLockResponse
	select {
	case msg := <-stream.sendCh:
		if msg.MessageType != signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_ACQUIRED {
			t.Fatalf("want ACQUIRED, got %v", msg.MessageType)
		}
		acquired = msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ACQUIRED")
	}

	// Send heartbeat with a longer TTL.
	stream.sendHeartbeat(60)

	select {
	case msg := <-stream.sendCh:
		if msg.MessageType != signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_TTL_EXTENDED {
			t.Errorf("want TTL_EXTENDED, got %v", msg.MessageType)
		}
		if !msg.ExpiresAt.AsTime().After(acquired.ExpiresAt.AsTime()) {
			t.Errorf("extended expiry %v must be after original %v",
				msg.ExpiresAt.AsTime(), acquired.ExpiresAt.AsTime())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for TTL_EXTENDED")
	}

	cancel()
}

func TestAcquireRestartLock_QueuePositionAndAcquire(t *testing.T) {
	srv, ls, lm := newLockServer()

	// First holder acquires immediately.
	holder1Ctx, holder1Cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	holder1Stream := newFakeLockStream(holder1Ctx)
	holder1Stream.sendInit("ns", "svc", 30)
	go srv.AcquireRestartLock(holder1Stream) //nolint:errcheck

	select {
	case msg := <-holder1Stream.sendCh:
		if msg.MessageType != signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_ACQUIRED {
			t.Fatalf("holder1: want ACQUIRED, got %v", msg.MessageType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("holder1: timed out waiting for ACQUIRED")
	}

	// Second caller should receive QUEUE_POSITION(1).
	waiter2Ctx, waiter2Cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	defer waiter2Cancel()
	waiter2Stream := newFakeLockStream(waiter2Ctx)
	waiter2Stream.sendInit("ns", "svc", 30)
	done2 := make(chan error, 1)
	go func() { done2 <- srv.AcquireRestartLock(waiter2Stream) }()

	select {
	case msg := <-waiter2Stream.sendCh:
		if msg.MessageType != signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_QUEUE_POSITION {
			t.Fatalf("waiter2: want QUEUE_POSITION, got %v", msg.MessageType)
		}
		if msg.Position != 1 {
			t.Errorf("waiter2: want position=1, got %d", msg.Position)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter2: timed out waiting for QUEUE_POSITION")
	}

	// Release holder1 — waiter2 should receive ACQUIRED.
	holder1Cancel()
	// Give the release a moment to propagate through the DB and notify.
	ls.forceRelease("ns", "svc")
	lm.NotifyFirst("ns", "svc")

	select {
	case msg := <-waiter2Stream.sendCh:
		if msg.MessageType != signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_ACQUIRED {
			t.Errorf("waiter2: want ACQUIRED after release, got %v", msg.MessageType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter2: timed out waiting for ACQUIRED after holder1 released")
	}
}

func TestAcquireRestartLock_QueuePositionDecrementsWhenAheadCancels(t *testing.T) {
	srv, ls, lm := newLockServer()

	// Holder acquires the lock.
	holderCtx, holderCancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	holderStream := newFakeLockStream(holderCtx)
	holderStream.sendInit("ns", "svc", 30)
	go srv.AcquireRestartLock(holderStream) //nolint:errcheck
	select {
	case <-holderStream.sendCh: // consume ACQUIRED
	case <-time.After(2 * time.Second):
		t.Fatal("holder: timed out")
	}

	// Waiter 1 joins at position 1.
	w1Ctx, w1Cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	w1Stream := newFakeLockStream(w1Ctx)
	w1Stream.sendInit("ns", "svc", 30)
	go srv.AcquireRestartLock(w1Stream) //nolint:errcheck
	select {
	case msg := <-w1Stream.sendCh:
		if msg.Position != 1 {
			t.Fatalf("w1: want position=1, got %d", msg.Position)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("w1: timed out waiting for QUEUE_POSITION")
	}

	// Waiter 2 joins at position 2.
	w2Ctx, w2Cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	defer w2Cancel()
	w2Stream := newFakeLockStream(w2Ctx)
	w2Stream.sendInit("ns", "svc", 30)
	go srv.AcquireRestartLock(w2Stream) //nolint:errcheck
	select {
	case msg := <-w2Stream.sendCh:
		if msg.Position != 2 {
			t.Fatalf("w2: want position=2, got %d", msg.Position)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("w2: timed out waiting for QUEUE_POSITION")
	}

	// Waiter 1 cancels — waiter 2 should shift to position 1.
	w1Cancel()
	select {
	case msg := <-w2Stream.sendCh:
		if msg.MessageType != signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_QUEUE_POSITION {
			t.Fatalf("w2: want QUEUE_POSITION after w1 cancel, got %v", msg.MessageType)
		}
		if msg.Position != 1 {
			t.Errorf("w2: want position=1 after w1 left, got %d", msg.Position)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("w2: timed out waiting for position update after w1 cancelled")
	}

	holderCancel()
	ls.forceRelease("ns", "svc")
	lm.NotifyFirst("ns", "svc")
}

func TestAcquireRestartLock_ContextCancelWhileWaiting(t *testing.T) {
	srv, _, _ := newLockServer()

	// Holder acquires the lock.
	holderCtx, holderCancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	defer holderCancel()
	holderStream := newFakeLockStream(holderCtx)
	holderStream.sendInit("ns", "svc", 30)
	go srv.AcquireRestartLock(holderStream) //nolint:errcheck
	select {
	case <-holderStream.sendCh: // consume ACQUIRED
	case <-time.After(2 * time.Second):
		t.Fatal("holder: timed out")
	}

	// Waiter joins then immediately cancels.
	waiterCtx, waiterCancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	waiterStream := newFakeLockStream(waiterCtx)
	waiterStream.sendInit("ns", "svc", 30)
	done := make(chan error, 1)
	go func() { done <- srv.AcquireRestartLock(waiterStream) }()
	// Drain initial QUEUE_POSITION.
	select {
	case <-waiterStream.sendCh:
	case <-time.After(2 * time.Second):
		t.Fatal("waiter: timed out waiting for QUEUE_POSITION")
	}

	waiterCancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("want context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for waiter goroutine to exit")
	}
}

func TestAcquireRestartLock_ReleasedOnStreamClose(t *testing.T) {
	srv, ls, _ := newLockServer()
	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	stream := newFakeLockStream(ctx)
	stream.sendInit("ns", "svc", 30)

	done := make(chan error, 1)
	go func() { done <- srv.AcquireRestartLock(stream) }()

	select {
	case <-stream.sendCh: // consume ACQUIRED
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ACQUIRED")
	}

	// Closing the stream (cancel context) should release the DB record.
	cancel()
	<-done

	// The lock should now be acquirable by a new holder.
	ctx2, cancel2 := context.WithCancel(spiffeCtx("spiffe://x/y"))
	defer cancel2()
	stream2 := newFakeLockStream(ctx2)
	stream2.sendInit("ns", "svc", 30)

	// Wait briefly for the release to propagate.
	time.Sleep(10 * time.Millisecond)
	_ = ls // forceRelease was called via Release in holdLock defer

	done2 := make(chan error, 1)
	go func() { done2 <- srv.AcquireRestartLock(stream2) }()
	select {
	case msg := <-stream2.sendCh:
		if msg.MessageType != signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_ACQUIRED {
			t.Errorf("want ACQUIRED for new holder, got %v", msg.MessageType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out: lock was not released when first stream closed")
	}
	cancel2()
	<-done2
}
