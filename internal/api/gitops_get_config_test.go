package api

import (
	"context"
	"encoding/json"
	"testing"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/bytepunx/signet/internal/auth"
	"github.com/bytepunx/signet/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeGetConfigStore implements gitopsStore for GetServiceConfig tests. Every
// method other than GetServiceConfig is a no-op stub — none of the other
// GitOpsServer RPCs are under test here.
type fakeGetConfigStore struct {
	getFn  func(ctx context.Context, namespace, service string) (json.RawMessage, int, error)
	called bool
}

func (f *fakeGetConfigStore) PutSOPSKey(context.Context, *store.SOPSKey) error { return nil }
func (f *fakeGetConfigStore) GetActiveSOPSKey(context.Context, string) (*store.SOPSKey, error) {
	return nil, store.ErrNotFound
}
func (f *fakeGetConfigStore) ListSOPSKeys(context.Context, string) ([]store.SOPSKey, error) {
	return nil, nil
}
func (f *fakeGetConfigStore) DeactivateSOPSKey(context.Context, string) error        { return nil }
func (f *fakeGetConfigStore) DeleteSOPSKey(context.Context, string) error            { return nil }
func (f *fakeGetConfigStore) PutRepository(context.Context, *store.Repository) error { return nil }
func (f *fakeGetConfigStore) GetRepository(context.Context, string) (*store.Repository, error) {
	return nil, store.ErrNotFound
}
func (f *fakeGetConfigStore) ListRepositories(context.Context) ([]store.Repository, error) {
	return nil, nil
}
func (f *fakeGetConfigStore) DeleteRepository(context.Context, string) error { return nil }
func (f *fakeGetConfigStore) PatchServiceConfig(context.Context, string, string, func(json.RawMessage) (json.RawMessage, error)) (int, error) {
	return 0, nil
}

func (f *fakeGetConfigStore) GetServiceConfig(ctx context.Context, namespace, service string) (json.RawMessage, int, error) {
	f.called = true
	return f.getFn(ctx, namespace, service)
}
func (f *fakeGetConfigStore) PutServiceConfigIfVersion(context.Context, string, string, json.RawMessage, int) (int, error) {
	return 0, nil
}

func TestGetServiceConfig_RequiresNamespaceAndService(t *testing.T) {
	fs := &fakeGetConfigStore{}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.GetServiceConfig(bearerCtx("tok"), &adminv1.GetServiceConfigRequest{Service: "svc"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.False(t, fs.called)
}

// TestGetServiceConfig_BearerTokenSuccess is the end-to-end happy path for
// the admin-token auth mode: no checker.Allow call, store called with the
// exact namespace/service, response carries content + version, and the read
// is audited under the admin's resolved identity.
func TestGetServiceConfig_BearerTokenSuccess(t *testing.T) {
	rec := &fakeRecorder{}
	checkerCalled := false
	fs := &fakeGetConfigStore{getFn: func(ctx context.Context, namespace, service string) (json.RawMessage, int, error) {
		assert.Equal(t, "authstar", namespace)
		assert.Equal(t, "portcullis", service)
		return json.RawMessage(`{"tenants":{"acme":{"sessionKeyGenerations":[{"version":1},{"version":2}]}}}`), 7, nil
	}}
	srv := &GitOpsServer{
		store:     fs,
		validator: &fakeTokenValidator{identity: "system:serviceaccount:signet:signet-admin"},
		checker:   &scopeCheckerFunc{allow: func(context.Context, string, string, string, string, string) error { checkerCalled = true; return nil }},
		audit:     rec,
	}

	resp, err := srv.GetServiceConfig(bearerCtx("tok"), &adminv1.GetServiceConfigRequest{
		Namespace: "authstar",
		Service:   "portcullis",
	})
	require.NoError(t, err)
	assert.EqualValues(t, 7, resp.GetVersion())
	assert.True(t, fs.called)
	assert.False(t, checkerCalled, "admin bearer-token auth must not consult the policy checker")

	gens := resp.GetContent().GetStructValue().GetFields()["tenants"].GetStructValue().
		GetFields()["acme"].GetStructValue().GetFields()["sessionKeyGenerations"].GetListValue().GetValues()
	require.Len(t, gens, 2)
	assert.EqualValues(t, 1, gens[0].GetStructValue().GetFields()["version"].GetNumberValue())
	assert.EqualValues(t, 2, gens[1].GetStructValue().GetFields()["version"].GetNumberValue())

	entry := rec.last()
	assert.Equal(t, "system:serviceaccount:signet:signet-admin", entry.SPIFFEID)
	assert.Equal(t, "get_config", entry.Action)
	assert.Equal(t, "authstar", entry.Namespace)
	assert.Equal(t, "portcullis/"+configAuditName, entry.SecretName)
	assert.Equal(t, "permitted", entry.Outcome)
}

// TestGetServiceConfig_SPIFFEOwnNamespacePasses verifies the SPIFFE path
// consults checker.Allow with the "get" permission — distinct from
// PatchServiceConfig's "put", per authorizeGitOpsRead's own doc comment.
func TestGetServiceConfig_SPIFFEOwnNamespacePasses(t *testing.T) {
	var gotPermission, gotNS, gotSvc string
	fs := &fakeGetConfigStore{getFn: func(ctx context.Context, namespace, service string) (json.RawMessage, int, error) {
		return json.RawMessage(`{}`), 1, nil
	}}
	srv := &GitOpsServer{
		store:     fs,
		validator: &fakeTokenValidator{},
		checker: &scopeCheckerFunc{allow: func(_ context.Context, spiffeID, permission, namespace, service, _ string) error {
			gotPermission, gotNS, gotSvc = permission, namespace, service
			return nil
		}},
	}

	_, err := srv.GetServiceConfig(spiffeCtx("spiffe://cluster.local/ns/authstar/sa/tower"), &adminv1.GetServiceConfigRequest{
		Namespace: "authstar",
		Service:   "portcullis",
	})
	require.NoError(t, err)
	assert.Equal(t, "get", gotPermission)
	assert.Equal(t, "authstar", gotNS)
	assert.Equal(t, "portcullis", gotSvc)
}

// TestGetServiceConfig_SPIFFEDeniedByChecker verifies a denied checker
// rejects the call before the store is ever consulted.
func TestGetServiceConfig_SPIFFEDeniedByChecker(t *testing.T) {
	fs := &fakeGetConfigStore{getFn: func(context.Context, string, string) (json.RawMessage, int, error) {
		t.Fatal("store must not be consulted when checker denies")
		return nil, 0, nil
	}}
	srv := &GitOpsServer{
		store:     fs,
		validator: &fakeTokenValidator{},
		checker: &scopeCheckerFunc{allow: func(context.Context, string, string, string, string, string) error {
			return auth.ErrUnauthorized
		}},
	}

	_, err := srv.GetServiceConfig(spiffeCtx("spiffe://cluster.local/ns/other/sa/whoever"), &adminv1.GetServiceConfigRequest{
		Namespace: "authstar",
		Service:   "portcullis",
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGetServiceConfig_NotFound(t *testing.T) {
	fs := &fakeGetConfigStore{getFn: func(context.Context, string, string) (json.RawMessage, int, error) {
		return nil, 0, store.ErrNotFound
	}}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.GetServiceConfig(bearerCtx("tok"), &adminv1.GetServiceConfigRequest{
		Namespace: "authstar",
		Service:   "portcullis",
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}
