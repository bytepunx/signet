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
	"google.golang.org/protobuf/types/known/structpb"
)

// fakePatchStore implements gitopsStore for PatchServiceConfig tests. Every
// method other than PatchServiceConfig is a no-op stub — none of the other
// GitOpsServer RPCs are under test here.
type fakePatchStore struct {
	patchFn func(ctx context.Context, namespace, service string, apply func(current json.RawMessage) (json.RawMessage, error)) (int, error)
	called  bool
}

func (f *fakePatchStore) PutSOPSKey(context.Context, *store.SOPSKey) error { return nil }
func (f *fakePatchStore) GetActiveSOPSKey(context.Context, string) (*store.SOPSKey, error) {
	return nil, store.ErrNotFound
}
func (f *fakePatchStore) ListSOPSKeys(context.Context, string) ([]store.SOPSKey, error) {
	return nil, nil
}
func (f *fakePatchStore) DeactivateSOPSKey(context.Context, string) error        { return nil }
func (f *fakePatchStore) DeleteSOPSKey(context.Context, string) error            { return nil }
func (f *fakePatchStore) PutRepository(context.Context, *store.Repository) error { return nil }
func (f *fakePatchStore) GetRepository(context.Context, string) (*store.Repository, error) {
	return nil, store.ErrNotFound
}
func (f *fakePatchStore) ListRepositories(context.Context) ([]store.Repository, error) {
	return nil, nil
}
func (f *fakePatchStore) DeleteRepository(context.Context, string) error { return nil }

func (f *fakePatchStore) PatchServiceConfig(ctx context.Context, namespace, service string, apply func(current json.RawMessage) (json.RawMessage, error)) (int, error) {
	f.called = true
	return f.patchFn(ctx, namespace, service, apply)
}

// GetServiceConfig and PutServiceConfigIfVersion aren't under test in this
// file (see gitops_get_config_test.go and gitops_put_config_test.go) --
// no-op stubs, same as every other non-PatchServiceConfig method above.
func (f *fakePatchStore) GetServiceConfig(context.Context, string, string) (json.RawMessage, int, error) {
	return nil, 0, store.ErrNotFound
}
func (f *fakePatchStore) PutServiceConfigIfVersion(context.Context, string, string, json.RawMessage, int) (int, error) {
	return 0, nil
}

func addOp(path string, value any) *adminv1.JsonPatchOperation {
	v, err := structpb.NewValue(value)
	if err != nil {
		panic(err)
	}
	return &adminv1.JsonPatchOperation{Op: "add", Path: path, Value: v}
}

// --- decodeJSONPatch ---

func TestDecodeJSONPatch_AppendsToArray(t *testing.T) {
	patch, err := decodeJSONPatch([]*adminv1.JsonPatchOperation{
		addOp("/tenants/acme/gens/-", float64(2)),
	})
	require.NoError(t, err)

	out, err := patch.Apply([]byte(`{"tenants":{"acme":{"gens":[1]}}}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"tenants":{"acme":{"gens":[1,2]}}}`, string(out))
}

func TestDecodeJSONPatch_RemoveOperationNeedsNoValue(t *testing.T) {
	patch, err := decodeJSONPatch([]*adminv1.JsonPatchOperation{
		{Op: "remove", Path: "/a"},
	})
	require.NoError(t, err)

	out, err := patch.Apply([]byte(`{"a":1,"b":2}`))
	require.NoError(t, err)
	assert.JSONEq(t, `{"b":2}`, string(out))
}

// --- PatchServiceConfig RPC ---

func TestPatchServiceConfig_RequiresNamespaceAndService(t *testing.T) {
	fs := &fakePatchStore{}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.PatchServiceConfig(bearerCtx("tok"), &adminv1.PatchServiceConfigRequest{
		Service:    "svc",
		Operations: []*adminv1.JsonPatchOperation{addOp("/a", 1)},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.False(t, fs.called)
}

func TestPatchServiceConfig_RequiresOperations(t *testing.T) {
	fs := &fakePatchStore{}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.PatchServiceConfig(bearerCtx("tok"), &adminv1.PatchServiceConfigRequest{
		Namespace: "ns", Service: "svc",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.False(t, fs.called)
}

// TestPatchServiceConfig_BearerTokenSuccess is the end-to-end happy path for
// the admin-token auth mode: no checker.Allow call, store called with the
// exact namespace/service, response carries the new version, and the write
// is audited under the admin's resolved identity.
func TestPatchServiceConfig_BearerTokenSuccess(t *testing.T) {
	rec := &fakeRecorder{}
	checkerCalled := false
	fs := &fakePatchStore{patchFn: func(ctx context.Context, namespace, service string, apply func(json.RawMessage) (json.RawMessage, error)) (int, error) {
		assert.Equal(t, "authstar", namespace)
		assert.Equal(t, "portcullis", service)
		out, err := apply(json.RawMessage(`{"tenants":{"acme":{"gens":[1]}}}`))
		require.NoError(t, err)
		assert.JSONEq(t, `{"tenants":{"acme":{"gens":[1,2]}}}`, string(out))
		return 5, nil
	}}
	srv := &GitOpsServer{
		store:     fs,
		validator: &fakeTokenValidator{identity: "system:serviceaccount:signet:signet-admin"},
		checker:   &scopeCheckerFunc{allow: func(context.Context, string, string, string, string, string) error { checkerCalled = true; return nil }},
		audit:     rec,
		bus:       NewBus(),
	}

	resp, err := srv.PatchServiceConfig(bearerCtx("tok"), &adminv1.PatchServiceConfigRequest{
		Namespace:  "authstar",
		Service:    "portcullis",
		Operations: []*adminv1.JsonPatchOperation{addOp("/tenants/acme/gens/-", float64(2))},
	})
	require.NoError(t, err)
	assert.EqualValues(t, 5, resp.GetVersion())
	assert.True(t, fs.called)
	assert.False(t, checkerCalled, "admin bearer-token auth must not consult the policy checker")

	entry := rec.last()
	assert.Equal(t, "system:serviceaccount:signet:signet-admin", entry.SPIFFEID)
	assert.Equal(t, "patch_config", entry.Action)
	assert.Equal(t, "authstar", entry.Namespace)
	assert.Equal(t, "portcullis/"+configAuditName, entry.SecretName)
	assert.Equal(t, "permitted", entry.Outcome)
}

// TestPatchServiceConfig_SPIFFEOwnNamespacePasses verifies the SPIFFE path
// consults checker.Allow with the "put" permission, matching SyncBundle's
// validateBundleScope convention exactly (bytepunx/signet#23).
func TestPatchServiceConfig_SPIFFEOwnNamespacePasses(t *testing.T) {
	var gotPermission, gotNS, gotSvc string
	fs := &fakePatchStore{patchFn: func(ctx context.Context, namespace, service string, apply func(json.RawMessage) (json.RawMessage, error)) (int, error) {
		return 1, nil
	}}
	srv := &GitOpsServer{
		store:     fs,
		validator: &fakeTokenValidator{},
		checker: &scopeCheckerFunc{allow: func(_ context.Context, spiffeID, permission, namespace, service, _ string) error {
			gotPermission, gotNS, gotSvc = permission, namespace, service
			if namespace == "authstar" && service == "tower" {
				return nil
			}
			return auth.ErrUnauthorized
		}},
		bus: NewBus(),
	}

	ctx := spiffeCtx("spiffe://cluster.local/ns/authstar/sa/tower")
	_, err := srv.PatchServiceConfig(ctx, &adminv1.PatchServiceConfigRequest{
		Namespace:  "authstar",
		Service:    "tower",
		Operations: []*adminv1.JsonPatchOperation{addOp("/a", 1)},
	})
	require.NoError(t, err)
	assert.Equal(t, "put", gotPermission)
	assert.Equal(t, "authstar", gotNS)
	assert.Equal(t, "tower", gotSvc)
	assert.True(t, fs.called)
}

// TestPatchServiceConfig_SPIFFECrossServiceRejected verifies a SPIFFE
// identity denied by the policy checker never reaches the store at all.
func TestPatchServiceConfig_SPIFFECrossServiceRejected(t *testing.T) {
	fs := &fakePatchStore{patchFn: func(context.Context, string, string, func(json.RawMessage) (json.RawMessage, error)) (int, error) {
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
	_, err := srv.PatchServiceConfig(ctx, &adminv1.PatchServiceConfigRequest{
		Namespace:  "authstar",
		Service:    "portcullis",
		Operations: []*adminv1.JsonPatchOperation{addOp("/a", 1)},
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.False(t, fs.called)
}

func TestPatchServiceConfig_ConflictMapsToAborted(t *testing.T) {
	fs := &fakePatchStore{patchFn: func(context.Context, string, string, func(json.RawMessage) (json.RawMessage, error)) (int, error) {
		return 0, store.ErrConflict
	}}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.PatchServiceConfig(bearerCtx("tok"), &adminv1.PatchServiceConfigRequest{
		Namespace: "ns", Service: "svc", Operations: []*adminv1.JsonPatchOperation{addOp("/a", 1)},
	})
	require.Error(t, err)
	assert.Equal(t, codes.Aborted, status.Code(err))
}

func TestPatchServiceConfig_NotFoundMapsToNotFound(t *testing.T) {
	fs := &fakePatchStore{patchFn: func(context.Context, string, string, func(json.RawMessage) (json.RawMessage, error)) (int, error) {
		return 0, store.ErrNotFound
	}}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.PatchServiceConfig(bearerCtx("tok"), &adminv1.PatchServiceConfigRequest{
		Namespace: "ns", Service: "svc", Operations: []*adminv1.JsonPatchOperation{addOp("/a", 1)},
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// TestPatchServiceConfig_MalformedPatchMapsToInvalidArgument verifies a
// patch that fails to apply (path doesn't exist) surfaces as the caller's
// mistake, not a server error, and never reaches the checker/write path
// beyond the store's own atomic apply attempt.
func TestPatchServiceConfig_MalformedPatchMapsToInvalidArgument(t *testing.T) {
	fs := &fakePatchStore{patchFn: func(ctx context.Context, namespace, service string, apply func(json.RawMessage) (json.RawMessage, error)) (int, error) {
		_, err := apply(json.RawMessage(`{"a":1}`))
		return 0, err
	}}
	srv := &GitOpsServer{store: fs, validator: &fakeTokenValidator{}}

	_, err := srv.PatchServiceConfig(bearerCtx("tok"), &adminv1.PatchServiceConfigRequest{
		Namespace: "ns", Service: "svc",
		// "/missing/deep/path" doesn't exist in {"a":1} — replace must fail.
		Operations: []*adminv1.JsonPatchOperation{{Op: "replace", Path: "/missing/deep/path", Value: mustValue(1)}},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func mustValue(v any) *structpb.Value {
	val, err := structpb.NewValue(v)
	if err != nil {
		panic(err)
	}
	return val
}
