package api

import (
	"context"
	"fmt"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/bytepunx/signet/internal/auth"
	"github.com/bytepunx/signet/internal/unseal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AdminServer implements adminv1.AdminServiceServer.
// Every method requires a valid Kubernetes SA bearer token in the gRPC metadata.
type AdminServer struct {
	adminv1.UnimplementedAdminServiceServer
	manager   unsealMgr
	validator tokenChecker
}

// NewAdminServer constructs an AdminServer.
func NewAdminServer(manager unsealMgr, validator tokenChecker) *AdminServer {
	return &AdminServer{manager: manager, validator: validator}
}

// requireToken extracts and validates the bearer SA token from gRPC metadata.
func (s *AdminServer) requireToken(ctx context.Context) error {
	token, err := auth.TokenFromMetadata(ctx)
	if err != nil {
		return toGRPCError(err)
	}
	if err := s.validator.Validate(ctx, token); err != nil {
		return toGRPCError(err)
	}
	return nil
}

// UnsealKey unseals the server with a direct 32-byte master key.
func (s *AdminServer) UnsealKey(ctx context.Context, req *adminv1.UnsealKeyRequest) (*adminv1.UnsealKeyResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	if len(req.Key) == 0 {
		return nil, status.Error(codes.InvalidArgument, "key must not be empty")
	}
	if err := s.manager.UnsealWithKey(req.Key); err != nil {
		return nil, toGRPCError(err)
	}
	st := s.manager.Status()
	return &adminv1.UnsealKeyResponse{
		Unsealed:       st.State == unseal.StateUnsealed,
		SharesReceived: int32(st.SharesReceived),
		SharesRequired: int32(st.SharesRequired),
		Message:        "unsealed successfully",
	}, nil
}

// UnsealShare submits one Shamir share. The server unseals when the threshold is met.
func (s *AdminServer) UnsealShare(ctx context.Context, req *adminv1.UnsealShareRequest) (*adminv1.UnsealShareResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	if len(req.Share) == 0 {
		return nil, status.Error(codes.InvalidArgument, "share must not be empty")
	}
	st, err := s.manager.SubmitShare(req.Share)
	if err != nil {
		return nil, toGRPCError(err)
	}

	msg := fmt.Sprintf("share accepted (%d/%d)", st.SharesReceived, st.SharesRequired)
	if st.State == unseal.StateUnsealed {
		msg = "unsealed successfully"
	}
	return &adminv1.UnsealShareResponse{
		Unsealed:       st.State == unseal.StateUnsealed,
		SharesReceived: int32(st.SharesReceived),
		SharesRequired: int32(st.SharesRequired),
		Message:        msg,
	}, nil
}

// Seal immediately wipes the master key from memory and transitions to sealed.
func (s *AdminServer) Seal(ctx context.Context, _ *adminv1.SealRequest) (*adminv1.SealResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	s.manager.Seal()
	return &adminv1.SealResponse{Message: "sealed"}, nil
}

// Status returns the current seal state and Shamir share progress.
func (s *AdminServer) Status(ctx context.Context, _ *adminv1.StatusRequest) (*adminv1.StatusResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	st := s.manager.Status()
	return &adminv1.StatusResponse{
		State:          toProtoState(st.State),
		SharesReceived: int32(st.SharesReceived),
		SharesRequired: int32(st.SharesRequired),
	}, nil
}

func toProtoState(s unseal.State) adminv1.StatusResponse_State {
	switch s {
	case unseal.StateSealed:
		return adminv1.StatusResponse_STATE_SEALED
	case unseal.StateUnsealing:
		return adminv1.StatusResponse_STATE_UNSEALING
	case unseal.StateUnsealed:
		return adminv1.StatusResponse_STATE_UNSEALED
	default:
		return adminv1.StatusResponse_STATE_UNSPECIFIED
	}
}
