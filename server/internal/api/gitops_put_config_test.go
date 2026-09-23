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

// fakePutConfigStore implements gitopsStore for PutServiceConfig tests.
// Every method other than PutServiceConfigIfVersion is a no-op stub — none
// of the other GitOpsServer RPCs are under test here.
type fakePutConfigStore struct {
	putFn  func(ctx context.Context, namespace, service string, content json.RawMessage, expectedVersion int) (int, error)
	called bool
}

func (f *fakePutConfigStore) PutSOPSKey(context.Context, *store.SOPSKey) error { return nil }
func (f *fakePutConfigStore) GetActiveSOPSKey(context.Context, string) (*store.SOPSKey, error) {
	return nil, store.ErrNotFound
}
func (f *fakePutConfigStore) ListSOPSKeys(context.Context, string) ([]store.SOPSKey, error) {
	return nil, nil
}
func (f *fakePutConfigStore) DeactivateSOPSKey(context.Context, string) error        { return nil }
func (f *fakePutConfigStore) DeleteSOPSKey(context.Context, string) error            { return nil }
func (f *fakePutConfigStore) PutRepository(context.Context, *store.Repository) error { return nil }
func (f *fakePutConfigStore) GetRepository(context.Context, string) (*store.Repository, error) {
	return nil, store.ErrNotFound
}
func (f *fakePutConfigStore) ListRepositories(context.Context) ([]store.Repository, error) {
	return nil, nil
}
func (f *fakePutConfigStore) DeleteRepository(context.Context, string) error { return nil }
func (f *fakePutConfigStore) PatchServiceConfig(context.Context, string, string, func(json.RawMessage) (json.RawMessage, error)) (int, error) {
	return 0, nil
}
func (f *fakePutConfigStore) GetServiceConfig(context.Context, string, string) (json.RawMessage, int, error) {
	return nil, 0, store.ErrNotFound
}

func (f *fakePutConfigStore) PutServiceConfigIfVersion(ctx context.Context, namespace, service string, content json.RawMessage, expectedVersion int) (int, error) {
	f.called = true
	return f.putFn(ctx, namespace, service, content, expectedVersion)
}

func TestPutServiceConfig_RequiresNamespaceAndService(t *testing.T) {
	fs := &fakePutConfigStore{}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.PutServiceConfig(bearerCtx("tok"), &adminv1.PutServiceConfigRequest{
		Service: "svc", Content: mustValue(map[string]any{"a": 1}),
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.False(t, fs.called)
}

func TestPutServiceConfig_RequiresContent(t *testing.T) {
	fs := &fakePutConfigStore{}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.PutServiceConfig(bearerCtx("tok"), &adminv1.PutServiceConfigRequest{
		Namespace: "ns", Service: "svc",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.False(t, fs.called)
}

func TestPutServiceConfig_RejectsNegativeExpectedVersion(t *testing.T) {
	fs := &fakePutConfigStore{}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.PutServiceConfig(bearerCtx("tok"), &adminv1.PutServiceConfigRequest{
		Namespace: "ns", Service: "svc", Content: mustValue(map[string]any{"a": 1}), ExpectedVersion: -1,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.False(t, fs.called)
}

// TestPutServiceConfig_BearerTokenCreateSuccess is the end-to-end happy path
// for the admin-token auth mode's create-only case: no checker.Allow call,
// store called with the exact namespace/service/content/expected_version,
// response carries the new version, and the write is audited under the
// admin's resolved identity with the distinct put_config_direct action.
func TestPutServiceConfig_BearerTokenCreateSuccess(t *testing.T) {
	rec := &fakeRecorder{}
	checkerCalled := false
	fs := &fakePutConfigStore{putFn: func(ctx context.Context, namespace, service string, content json.RawMessage, expectedVersion int) (int, error) {
		assert.Equal(t, "authstar", namespace)
		assert.Equal(t, "keep", service)
		assert.JSONEq(t, `{"stripeKey":"sk_live_x"}`, string(content))
		assert.Equal(t, 0, expectedVersion)
		return 1, nil
	}}
	srv := &GitOpsServer{
		store:     fs,
		validator: &fakeTokenValidator{identity: "system:serviceaccount:signet:signet-admin"},
		checker:   &scopeCheckerFunc{allow: func(context.Context, string, string, string, string, string) error { checkerCalled = true; return nil }},
		audit:     rec,
		bus:       NewBus(),
	}

	resp, err := srv.PutServiceConfig(bearerCtx("tok"), &adminv1.PutServiceConfigRequest{
		Namespace: "authstar",
		Service:   "keep",
		Content:   mustValue(map[string]any{"stripeKey": "sk_live_x"}),
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, resp.GetVersion())
	assert.True(t, fs.called)
	assert.False(t, checkerCalled, "admin bearer-token auth must not consult the policy checker")

	entry := rec.last()
	assert.Equal(t, "system:serviceaccount:signet:signet-admin", entry.SPIFFEID)
	assert.Equal(t, "put_config_direct", entry.Action)
	assert.Equal(t, "authstar", entry.Namespace)
	assert.Equal(t, "keep/"+configAuditName, entry.SecretName)
	assert.Equal(t, "permitted", entry.Outcome)
}

// TestPutServiceConfig_SPIFFEOwnNamespacePasses verifies the SPIFFE path
// consults checker.Allow with the "put" permission, matching
// SyncBundle/PatchServiceConfig's convention exactly — this is the Helm
// post-install-hook path (bytepunx/signet#80).
func TestPutServiceConfig_SPIFFEOwnNamespacePasses(t *testing.T) {
	var gotPermission, gotNS, gotSvc string
	fs := &fakePutConfigStore{putFn: func(context.Context, string, string, json.RawMessage, int) (int, error) {
		return 1, nil
	}}
	srv := &GitOpsServer{
		store:     fs,
		validator: &fakeTokenValidator{},
		checker: &scopeCheckerFunc{allow: func(_ context.Context, spiffeID, permission, namespace, service, _ string) error {
			gotPermission, gotNS, gotSvc = permission, namespace, service
			if namespace == "authstar" && service == "keep" {
				return nil
			}
			return auth.ErrUnauthorized
		}},
		bus: NewBus(),
	}

	ctx := spiffeCtx("spiffe://cluster.local/ns/authstar/sa/keep")
	_, err := srv.PutServiceConfig(ctx, &adminv1.PutServiceConfigRequest{
		Namespace: "authstar",
		Service:   "keep",
		Content:   mustValue(map[string]any{"a": 1}),
	})
	require.NoError(t, err)
	assert.Equal(t, "put", gotPermission)
	assert.Equal(t, "authstar", gotNS)
	assert.Equal(t, "keep", gotSvc)
	assert.True(t, fs.called)
}

// TestPutServiceConfig_SPIFFECrossServiceRejected verifies a SPIFFE identity
// denied by the policy checker never reaches the store at all.
func TestPutServiceConfig_SPIFFECrossServiceRejected(t *testing.T) {
	fs := &fakePutConfigStore{putFn: func(context.Context, string, string, json.RawMessage, int) (int, error) {
		t.Fatal("store must not be called when checker.Allow denies")
		return 0, nil
	}}
	srv := &GitOpsServer{
		store:     fs,
		validator: &fakeTokenValidator{},
		checker:   &scopeCheckerFunc{allow: func(context.Context, string, string, string, string, string) error { return auth.ErrUnauthorized }},
		bus:       NewBus(),
	}

	ctx := spiffeCtx("spiffe://cluster.local/ns/authstar/sa/tower")
	_, err := srv.PutServiceConfig(ctx, &adminv1.PutServiceConfigRequest{
		Namespace: "authstar",
		Service:   "portcullis",
		Content:   mustValue(map[string]any{"a": 1}),
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.False(t, fs.called)
}

// TestPutServiceConfig_AlreadyExistsMapsToAlreadyExists verifies a
// create-only call (expected_version=0) against an existing document
// surfaces as codes.AlreadyExists — the Helm-hook-re-run-safety guarantee.
func TestPutServiceConfig_AlreadyExistsMapsToAlreadyExists(t *testing.T) {
	fs := &fakePutConfigStore{putFn: func(context.Context, string, string, json.RawMessage, int) (int, error) {
		return 0, store.ErrAlreadyExists
	}}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.PutServiceConfig(bearerCtx("tok"), &adminv1.PutServiceConfigRequest{
		Namespace: "ns", Service: "svc", Content: mustValue(map[string]any{"a": 1}),
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func TestPutServiceConfig_ConflictMapsToAborted(t *testing.T) {
	fs := &fakePutConfigStore{putFn: func(context.Context, string, string, json.RawMessage, int) (int, error) {
		return 0, store.ErrConflict
	}}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.PutServiceConfig(bearerCtx("tok"), &adminv1.PutServiceConfigRequest{
		Namespace: "ns", Service: "svc", Content: mustValue(map[string]any{"a": 1}), ExpectedVersion: 3,
	})
	require.Error(t, err)
	assert.Equal(t, codes.Aborted, status.Code(err))
}

func TestPutServiceConfig_NotFoundMapsToNotFound(t *testing.T) {
	fs := &fakePutConfigStore{putFn: func(context.Context, string, string, json.RawMessage, int) (int, error) {
		return 0, store.ErrNotFound
	}}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.PutServiceConfig(bearerCtx("tok"), &adminv1.PutServiceConfigRequest{
		Namespace: "ns", Service: "svc", Content: mustValue(map[string]any{"a": 1}), ExpectedVersion: 3,
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}
