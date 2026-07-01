package api

import (
	"errors"

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
		return status.Error(codes.Internal, "internal error")
	}
}
