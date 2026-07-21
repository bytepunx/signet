package api

import (
	"errors"
	"log/slog"

	"github.com/bytepunx/signet/internal/auth"
	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
	"github.com/bytepunx/signet/internal/unseal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// toGRPCError maps internal sentinel errors to the appropriate gRPC status code.
// Unknown errors map to codes.Internal so callers never receive raw internal details.
func toGRPCError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, auth.ErrUnauthenticated), errors.Is(err, auth.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, auth.ErrUnauthorized):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, store.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, store.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, icrypto.ErrKeyNotSet):
		return status.Error(codes.Unavailable, "server is sealed")
	case errors.Is(err, icrypto.ErrAuthenticationFailed):
		return status.Error(codes.Internal, "decryption failed: data may be corrupt")
	case errors.Is(err, unseal.ErrAlreadyUnsealed):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, unseal.ErrShamirNotConfigured):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, unseal.ErrInvalidShare):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, unseal.ErrSharesExpired):
		return status.Error(codes.DeadlineExceeded, err.Error())
	case errors.Is(err, unseal.ErrInvalidConfig):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		// Every other branch above maps to a specific, already-understood
		// condition and returns a specific message; this default case is by
		// definition something unclassified, and previously vanished
		// entirely — the client saw only "internal error" and there was no
		// server-side trace at all to diagnose it from. Logging here is what
		// actually surfaced a real bug (a CockroachDB-incompatible query in
		// TryAcquireLock) during live smoke testing; without it, that bug
		// was completely silent on the server side.
		slog.Error("unmapped internal error", "err", err)
		return status.Error(codes.Internal, "internal error")
	}
}
