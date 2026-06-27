package api

import (
	"context"
	"fmt"
	"testing"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/bytepunx/signet/internal/auth"
	"github.com/bytepunx/signet/internal/unseal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// --- fakes ---

type fakeUnsealMgr struct {
	unsealKeyErr       error
	submitShareResult  unseal.Status
	submitShareErr     error
	statusResult       unseal.Status
	sealCalled         bool
}

func (f *fakeUnsealMgr) UnsealWithKey(_ []byte) error { return f.unsealKeyErr }
func (f *fakeUnsealMgr) SubmitShare(_ []byte) (unseal.Status, error) {
	return f.submitShareResult, f.submitShareErr
}
func (f *fakeUnsealMgr) Seal()                { f.sealCalled = true }
func (f *fakeUnsealMgr) Status() unseal.Status { return f.statusResult }

type fakeTokenChecker struct {
	err error
}

func (f *fakeTokenChecker) Validate(_ context.Context, _ string) error { return f.err }

// bearerCtx returns a context with an Authorization: Bearer <token> gRPC metadata header.
func bearerCtx(token string) context.Context {
	md := metadata.Pairs("authorization", fmt.Sprintf("Bearer %s", token))
	return metadata.NewIncomingContext(context.Background(), md)
}

// --- requireToken tests ---

func TestAdminServer_NoToken_Unauthenticated(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{})
	_, err := srv.Status(context.Background(), &adminv1.StatusRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("want Unauthenticated, got %v", err)
	}
}

func TestAdminServer_InvalidToken_Unauthenticated(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{err: auth.ErrInvalidToken})
	_, err := srv.Status(bearerCtx("bad-token"), &adminv1.StatusRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("want Unauthenticated, got %v", err)
	}
}

// --- UnsealKey tests ---

func TestUnsealKey_EmptyKey(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{})
	_, err := srv.UnsealKey(bearerCtx("tok"), &adminv1.UnsealKeyRequest{Key: nil})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestUnsealKey_AlreadyUnsealed(t *testing.T) {
	mgr := &fakeUnsealMgr{unsealKeyErr: unseal.ErrAlreadyUnsealed}
	srv := NewAdminServer(mgr, &fakeTokenChecker{})
	_, err := srv.UnsealKey(bearerCtx("tok"), &adminv1.UnsealKeyRequest{Key: []byte("k")})
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("want FailedPrecondition, got %v", err)
	}
}

func TestUnsealKey_Success(t *testing.T) {
	mgr := &fakeUnsealMgr{
		statusResult: unseal.Status{State: unseal.StateUnsealed},
	}
	srv := NewAdminServer(mgr, &fakeTokenChecker{})
	resp, err := srv.UnsealKey(bearerCtx("tok"), &adminv1.UnsealKeyRequest{Key: make([]byte, 32)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Unsealed {
		t.Error("expected Unsealed = true")
	}
}

// --- UnsealShare tests ---

func TestUnsealShare_EmptyShare(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{})
	_, err := srv.UnsealShare(bearerCtx("tok"), &adminv1.UnsealShareRequest{Share: nil})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestUnsealShare_InvalidShare(t *testing.T) {
	mgr := &fakeUnsealMgr{submitShareErr: unseal.ErrInvalidShare}
	srv := NewAdminServer(mgr, &fakeTokenChecker{})
	_, err := srv.UnsealShare(bearerCtx("tok"), &adminv1.UnsealShareRequest{Share: []byte("bad")})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestUnsealShare_SharesExpired(t *testing.T) {
	mgr := &fakeUnsealMgr{submitShareErr: unseal.ErrSharesExpired}
	srv := NewAdminServer(mgr, &fakeTokenChecker{})
	_, err := srv.UnsealShare(bearerCtx("tok"), &adminv1.UnsealShareRequest{Share: []byte("s")})
	if status.Code(err) != codes.DeadlineExceeded {
		t.Errorf("want DeadlineExceeded, got %v", err)
	}
}

func TestUnsealShare_PartialProgress(t *testing.T) {
	mgr := &fakeUnsealMgr{
		submitShareResult: unseal.Status{
			State:          unseal.StateUnsealing,
			SharesReceived: 1,
			SharesRequired: 3,
		},
	}
	srv := NewAdminServer(mgr, &fakeTokenChecker{})
	resp, err := srv.UnsealShare(bearerCtx("tok"), &adminv1.UnsealShareRequest{Share: []byte("s")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Unsealed {
		t.Error("expected Unsealed = false")
	}
	if resp.SharesReceived != 1 {
		t.Errorf("SharesReceived = %d, want 1", resp.SharesReceived)
	}
	if resp.SharesRequired != 3 {
		t.Errorf("SharesRequired = %d, want 3", resp.SharesRequired)
	}
}

func TestUnsealShare_ThresholdMet(t *testing.T) {
	mgr := &fakeUnsealMgr{
		submitShareResult: unseal.Status{
			State:          unseal.StateUnsealed,
			SharesReceived: 3,
			SharesRequired: 3,
		},
	}
	srv := NewAdminServer(mgr, &fakeTokenChecker{})
	resp, err := srv.UnsealShare(bearerCtx("tok"), &adminv1.UnsealShareRequest{Share: []byte("s")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Unsealed {
		t.Error("expected Unsealed = true")
	}
}

// --- Seal tests ---

func TestSeal_Success(t *testing.T) {
	mgr := &fakeUnsealMgr{}
	srv := NewAdminServer(mgr, &fakeTokenChecker{})
	resp, err := srv.Seal(bearerCtx("tok"), &adminv1.SealRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mgr.sealCalled {
		t.Error("expected Seal to be called on manager")
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestSeal_NoToken(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{})
	_, err := srv.Seal(context.Background(), &adminv1.SealRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("want Unauthenticated, got %v", err)
	}
}

// --- Status tests ---

func TestStatus_AllStates(t *testing.T) {
	tests := []struct {
		state     unseal.State
		wantProto adminv1.StatusResponse_State
	}{
		{unseal.StateSealed, adminv1.StatusResponse_STATE_SEALED},
		{unseal.StateUnsealing, adminv1.StatusResponse_STATE_UNSEALING},
		{unseal.StateUnsealed, adminv1.StatusResponse_STATE_UNSEALED},
	}
	for _, tc := range tests {
		t.Run(tc.state.String(), func(t *testing.T) {
			mgr := &fakeUnsealMgr{statusResult: unseal.Status{State: tc.state}}
			srv := NewAdminServer(mgr, &fakeTokenChecker{})
			resp, err := srv.Status(bearerCtx("tok"), &adminv1.StatusRequest{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.State != tc.wantProto {
				t.Errorf("state = %v, want %v", resp.State, tc.wantProto)
			}
		})
	}
}

func TestStatus_ShareProgress(t *testing.T) {
	mgr := &fakeUnsealMgr{
		statusResult: unseal.Status{
			State:          unseal.StateUnsealing,
			SharesReceived: 2,
			SharesRequired: 5,
		},
	}
	srv := NewAdminServer(mgr, &fakeTokenChecker{})
	resp, err := srv.Status(bearerCtx("tok"), &adminv1.StatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SharesReceived != 2 {
		t.Errorf("SharesReceived = %d, want 2", resp.SharesReceived)
	}
	if resp.SharesRequired != 5 {
		t.Errorf("SharesRequired = %d, want 5", resp.SharesRequired)
	}
}
