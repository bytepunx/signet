package api

import (
	"errors"
	"fmt"
	"testing"

	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/auth"
	"github.com/bytepunx/signet/internal/store"
	"github.com/bytepunx/signet/internal/unseal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGRPCError_Nil(t *testing.T) {
	if got := toGRPCError(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestToGRPCError_Mappings(t *testing.T) {
	tests := []struct {
		err      error
		wantCode codes.Code
	}{
		{auth.ErrUnauthenticated, codes.Unauthenticated},
		{auth.ErrInvalidToken, codes.Unauthenticated},
		{auth.ErrUnauthorized, codes.PermissionDenied},
		{store.ErrNotFound, codes.NotFound},
		{store.ErrInvalidInput, codes.InvalidArgument},
		{icrypto.ErrKeyNotSet, codes.Unavailable},
		{icrypto.ErrAuthenticationFailed, codes.Internal},
		{unseal.ErrAlreadyUnsealed, codes.FailedPrecondition},
		{unseal.ErrShamirNotConfigured, codes.FailedPrecondition},
		{unseal.ErrInvalidShare, codes.InvalidArgument},
		{unseal.ErrSharesExpired, codes.DeadlineExceeded},
		{unseal.ErrInvalidConfig, codes.FailedPrecondition},
		{errors.New("unexpected"), codes.Internal},
	}

	for _, tc := range tests {
		t.Run(tc.err.Error(), func(t *testing.T) {
			got := toGRPCError(tc.err)
			if got == nil {
				t.Fatal("expected non-nil error")
			}
			if code := status.Code(got); code != tc.wantCode {
				t.Errorf("code = %v, want %v", code, tc.wantCode)
			}
		})
	}
}

func TestToGRPCError_WrappedErrors(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", auth.ErrUnauthenticated)
	got := toGRPCError(wrapped)
	if status.Code(got) != codes.Unauthenticated {
		t.Errorf("wrapped ErrUnauthenticated: got %v, want Unauthenticated", status.Code(got))
	}

	wrapped = fmt.Errorf("outer: %w", store.ErrNotFound)
	got = toGRPCError(wrapped)
	if status.Code(got) != codes.NotFound {
		t.Errorf("wrapped ErrNotFound: got %v, want NotFound", status.Code(got))
	}
}

func TestToGRPCError_UnknownDoesNotLeakDetail(t *testing.T) {
	err := errors.New("internal db connection string with password")
	got := toGRPCError(err)
	msg := status.Convert(got).Message()
	if msg != "internal error" {
		t.Errorf("unexpected message leaked: %q", msg)
	}
}
