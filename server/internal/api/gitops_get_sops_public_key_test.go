package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/bytepunx/signet/internal/auth"
	"github.com/bytepunx/signet/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeSOPSKeyStore implements gitopsStore for GetSOPSPublicKey tests. Every
// method other than GetActiveSOPSKey is a no-op stub — none of the other
// GitOpsServer RPCs are under test here.
type fakeSOPSKeyStore struct {
	getFn  func(ctx context.Context, env string) (*store.SOPSKey, error)
	called bool
}

func (f *fakeSOPSKeyStore) PutSOPSKey(context.Context, *store.SOPSKey) error { return nil }
func (f *fakeSOPSKeyStore) GetActiveSOPSKey(ctx context.Context, env string) (*store.SOPSKey, error) {
	f.called = true
	return f.getFn(ctx, env)
}
func (f *fakeSOPSKeyStore) ListSOPSKeys(context.Context, string) ([]store.SOPSKey, error) {
	return nil, nil
}
func (f *fakeSOPSKeyStore) DeactivateSOPSKey(context.Context, string) error        { return nil }
func (f *fakeSOPSKeyStore) DeleteSOPSKey(context.Context, string) error            { return nil }
func (f *fakeSOPSKeyStore) PutRepository(context.Context, *store.Repository) error { return nil }
func (f *fakeSOPSKeyStore) GetRepository(context.Context, string) (*store.Repository, error) {
	return nil, store.ErrNotFound
}
func (f *fakeSOPSKeyStore) ListRepositories(context.Context) ([]store.Repository, error) {
	return nil, nil
}
func (f *fakeSOPSKeyStore) DeleteRepository(context.Context, string) error { return nil }
func (f *fakeSOPSKeyStore) PatchServiceConfig(context.Context, string, string, func(json.RawMessage) (json.RawMessage, error)) (int, error) {
	return 0, nil
}
func (f *fakeSOPSKeyStore) GetServiceConfig(context.Context, string, string) (json.RawMessage, int, error) {
	return nil, 0, store.ErrNotFound
}
func (f *fakeSOPSKeyStore) PutServiceConfigIfVersion(context.Context, string, string, json.RawMessage, int) (int, error) {
	return 0, nil
}

// TestGetSOPSPublicKey_BearerTokenSuccess is the admin-token happy path.
func TestGetSOPSPublicKey_BearerTokenSuccess(t *testing.T) {
	fs := &fakeSOPSKeyStore{getFn: func(ctx context.Context, env string) (*store.SOPSKey, error) {
		assert.Equal(t, "prod", env)
		return &store.SOPSKey{PublicKey: "age1abc", Environment: "prod", CreatedAt: time.Now()}, nil
	}}
	srv := &GitOpsServer{
		store:       fs,
		validator:   &fakeTokenValidator{identity: "system:serviceaccount:signet:signet-admin"},
		environment: "prod",
	}

	resp, err := srv.GetSOPSPublicKey(bearerCtx("tok"), &adminv1.GetSOPSPublicKeyRequest{})
	require.NoError(t, err)
	assert.True(t, fs.called)
	assert.Equal(t, "age1abc", resp.GetPublicKey())
	assert.Equal(t, "prod", resp.GetEnvironment())
}

// TestGetSOPSPublicKey_SPIFFEIdentityGranted verifies the SPIFFE mTLS path
// succeeds without any policy/checker consultation — unlike
// GetServiceConfig/SyncBundle/PatchServiceConfig, this RPC has no
// namespace/service to scope against, so any authenticated workload
// identity is sufficient, even one with no relationship at all to the
// "prod" environment's namespace conventions (bytepunx/signet#78).
func TestGetSOPSPublicKey_SPIFFEIdentityGranted(t *testing.T) {
	fs := &fakeSOPSKeyStore{getFn: func(context.Context, string) (*store.SOPSKey, error) {
		return &store.SOPSKey{PublicKey: "age1xyz", CreatedAt: time.Now()}, nil
	}}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	resp, err := srv.GetSOPSPublicKey(
		spiffeCtx("spiffe://cluster.local/ns/authstar/sa/herald"),
		&adminv1.GetSOPSPublicKeyRequest{},
	)
	require.NoError(t, err)
	assert.True(t, fs.called)
	assert.Equal(t, "age1xyz", resp.GetPublicKey())
}

// TestGetSOPSPublicKey_NoCredentialsRejected verifies a caller presenting
// neither a bearer token nor a verified SPIFFE identity is rejected before
// the store is ever consulted.
func TestGetSOPSPublicKey_NoCredentialsRejected(t *testing.T) {
	fs := &fakeSOPSKeyStore{getFn: func(context.Context, string) (*store.SOPSKey, error) {
		t.Fatal("store must not be consulted with no credentials presented")
		return nil, nil
	}}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.GetSOPSPublicKey(context.Background(), &adminv1.GetSOPSPublicKeyRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestGetSOPSPublicKey_InvalidBearerTokenRejected verifies a malformed/
// invalid bearer token fails closed rather than falling back to the SPIFFE
// path — mirroring authorizeGitOpsCall's documented behavior.
func TestGetSOPSPublicKey_InvalidBearerTokenRejected(t *testing.T) {
	fs := &fakeSOPSKeyStore{getFn: func(context.Context, string) (*store.SOPSKey, error) {
		t.Fatal("store must not be consulted when the bearer token is invalid")
		return nil, nil
	}}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{err: auth.ErrInvalidToken}}

	_, err := srv.GetSOPSPublicKey(bearerCtx("bad-token"), &adminv1.GetSOPSPublicKeyRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// TestGetSOPSPublicKey_NoActiveKeyNotFound verifies a store error (e.g. no
// active key provisioned yet) propagates as NotFound.
func TestGetSOPSPublicKey_NoActiveKeyNotFound(t *testing.T) {
	fs := &fakeSOPSKeyStore{getFn: func(context.Context, string) (*store.SOPSKey, error) {
		return nil, store.ErrNotFound
	}}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.GetSOPSPublicKey(
		spiffeCtx("spiffe://cluster.local/ns/authstar/sa/herald"),
		&adminv1.GetSOPSPublicKeyRequest{},
	)
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}
