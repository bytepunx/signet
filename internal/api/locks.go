package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	signetv1 "github.com/bytepunx/signet/gen/signet/v1"
	"github.com/bytepunx/signet/internal/auth"
	"github.com/bytepunx/signet/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// heartbeatMsg carries a heartbeat message's TTL extension request.
type heartbeatMsg struct{ ttlSeconds int32 }

// AcquireRestartLock is a bidirectional streaming handler that serialises rolling
// restarts for a given (namespace, service). Clients block here until the lock is
// granted, then hold it by keeping the stream open. Closing the stream releases
// the lock and unblocks the next waiter.
//
// Message flow:
//
//	client → server  : AcquireRestartLockRequest{namespace, service, ttl_seconds}
//	server → client  : QUEUE_POSITION{position} (repeated as position changes)
//	server → client  : ACQUIRED{token, expires_at}
//	client → server  : heartbeat=true  (to extend TTL while holding)
//	server → client  : TTL_EXTENDED{expires_at}
//	client closes stream → lock released
func (s *SecretsServer) AcquireRestartLock(stream grpc.BidiStreamingServer[signetv1.AcquireRestartLockRequest, signetv1.AcquireRestartLockResponse]) error {
	ctx := stream.Context()

	// --- 1. Read the initial request ---
	req, err := stream.Recv()
	if err != nil {
		return err
	}
	if req.Namespace == "" || req.Service == "" {
		return status.Error(codes.InvalidArgument, "namespace and service are required")
	}
	if req.TtlSeconds <= 0 {
		return status.Error(codes.InvalidArgument, "ttl_seconds must be > 0")
	}
	ns, svc := req.Namespace, req.Service
	ttl := time.Duration(req.TtlSeconds) * time.Second

	// --- 2. Auth ---
	spiffeID, authErr := auth.SPIFFEIDFromContext(ctx)
	if authErr != nil {
		return toGRPCError(authErr)
	}
	if err := s.checker.Allow(ctx, spiffeID, "lock", ns, svc, ""); err != nil {
		return toGRPCError(err)
	}

	// --- 3. Drain incoming messages into a heartbeat channel ---
	// The goroutine runs for the lifetime of the stream so heartbeats are not
	// buffered in the gRPC layer while the handler is in its select loop.
	hbCh := make(chan heartbeatMsg, 8)
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				return
			}
			if msg.Heartbeat {
				select {
				case hbCh <- heartbeatMsg{msg.TtlSeconds}:
				default:
				}
			}
		}
	}()

	// --- 4. Attempt immediate acquisition ---
	token, err := newLockToken()
	if err != nil {
		return status.Errorf(codes.Internal, "generate lock token: %v", err)
	}
	expiresAt, acquired, err := s.lockMgr.TryAcquire(ctx, ns, svc, token, ttl)
	if err != nil {
		return toGRPCError(err)
	}
	if acquired {
		return s.holdLock(ctx, stream, ns, svc, token, ttl, expiresAt, hbCh)
	}

	// --- 5. Join the wait queue ---
	waiter, pos := s.lockMgr.Enqueue(ns, svc)
	dequeued := false
	defer func() {
		if !dequeued {
			s.lockMgr.Dequeue(ns, svc, waiter)
		}
	}()

	// If we landed at position 1 there may have been a release between our
	// initial TryAcquire and Enqueue. Try again before blocking.
	if pos == 1 {
		expiresAt, acquired, err = s.lockMgr.TryAcquire(ctx, ns, svc, token, ttl)
		if err != nil {
			return toGRPCError(err)
		}
		if acquired {
			s.lockMgr.Dequeue(ns, svc, waiter)
			dequeued = true
			return s.holdLock(ctx, stream, ns, svc, token, ttl, expiresAt, hbCh)
		}
	}

	// Send initial queue position.
	if err := stream.Send(&signetv1.AcquireRestartLockResponse{
		MessageType: signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_QUEUE_POSITION,
		Position:    pos,
	}); err != nil {
		return err
	}

	// --- 6. Wait loop: block until notified, then race to acquire ---
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case newPos := <-waiter.posUpdates:
			if err := stream.Send(&signetv1.AcquireRestartLockResponse{
				MessageType: signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_QUEUE_POSITION,
				Position:    newPos,
			}); err != nil {
				return err
			}

		case <-waiter.notify:
			expiresAt, acquired, err = s.lockMgr.TryAcquire(ctx, ns, svc, token, ttl)
			if err != nil {
				return toGRPCError(err)
			}
			if acquired {
				// Remove from queue now so remaining waiters get updated positions.
				s.lockMgr.Dequeue(ns, svc, waiter)
				dequeued = true
				return s.holdLock(ctx, stream, ns, svc, token, ttl, expiresAt, hbCh)
			}
			// Lost the DB race (another signet instance's waiter won).
			// Stay at position 1; the sweeper will trigger NotifyFirst again.
		}
	}
}

// holdLock sends ACQUIRED then processes heartbeats until the stream closes.
// The lock is released (with local notification) when this function returns.
func (s *SecretsServer) holdLock(
	ctx context.Context,
	stream grpc.BidiStreamingServer[signetv1.AcquireRestartLockRequest, signetv1.AcquireRestartLockResponse],
	ns, svc, token string,
	originalTTL time.Duration,
	expiresAt time.Time,
	hbCh <-chan heartbeatMsg,
) error {
	defer s.lockMgr.Release(context.Background(), ns, svc, token)

	if err := stream.Send(&signetv1.AcquireRestartLockResponse{
		MessageType: signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_ACQUIRED,
		Token:       token,
		ExpiresAt:   timestamppb.New(expiresAt),
	}); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case hb := <-hbCh:
			extendBy := originalTTL
			if hb.ttlSeconds > 0 {
				extendBy = time.Duration(hb.ttlSeconds) * time.Second
			}
			newExpiry, err := s.lockMgr.Heartbeat(ctx, ns, svc, token, extendBy)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return status.Error(codes.Aborted, "lock expired before heartbeat arrived; re-acquire required")
				}
				return toGRPCError(err)
			}
			if err := stream.Send(&signetv1.AcquireRestartLockResponse{
				MessageType: signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_TTL_EXTENDED,
				ExpiresAt:   timestamppb.New(newExpiry),
			}); err != nil {
				return err
			}
		}
	}
}

// newLockToken generates a cryptographically random, URL-safe token.
func newLockToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
