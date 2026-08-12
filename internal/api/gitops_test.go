package api

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bytepunx/signet/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestSkippedToErrors verifies the formatting used to surface
// bytepunx/signet#22 skips through TriggerSyncResponse/SyncBundleResponse's
// Errors field: nil in, nil out (so an empty Errors list doesn't print an
// empty "Warnings:" block on the CLI side), and one formatted message per
// skipped path otherwise.
func TestSkippedToErrors(t *testing.T) {
	if got := skippedToErrors(nil); got != nil {
		t.Errorf("skippedToErrors(nil) = %v, want nil", got)
	}
	if got := skippedToErrors([]string{}); got != nil {
		t.Errorf("skippedToErrors(empty) = %v, want nil", got)
	}

	got := skippedToErrors([]string{"config/ns/svc/extra.yaml", "secrets/ns/svc/sub/key.yaml"})
	if len(got) != 2 {
		t.Fatalf("skippedToErrors returned %d messages, want 2", len(got))
	}
	for i, path := range []string{"config/ns/svc/extra.yaml", "secrets/ns/svc/sub/key.yaml"} {
		if !strings.Contains(got[i], path) {
			t.Errorf("message %d = %q, want it to mention path %q", i, got[i], path)
		}
	}
}

// fakeTokenValidator implements tokenChecker for SyncBundle auth tests.
type fakeTokenValidator struct {
	identity string
	err      error
}

func (f *fakeTokenValidator) Validate(context.Context, string) (string, error) {
	return f.identity, f.err
}

// scopeCheckerFunc implements permissionChecker for SyncBundle scope tests.
type scopeCheckerFunc struct {
	allow func(ctx context.Context, spiffeID, permission, namespace, service, secretName string) error
}

func (f *scopeCheckerFunc) Allow(ctx context.Context, spiffeID, permission, namespace, service, secretName string) error {
	return f.allow(ctx, spiffeID, permission, namespace, service, secretName)
}

// --- authorizeSyncBundle ---

func TestAuthorizeSyncBundle_ValidBearerTokenGrantsUnscopedAccess(t *testing.T) {
	srv := &GitOpsServer{validator: &fakeTokenValidator{identity: "system:serviceaccount:signet:signet-admin"}}
	actor, scoped, err := srv.authorizeSyncBundle(bearerCtx("good-token"))
	require.NoError(t, err)
	assert.False(t, scoped, "bearer-token auth must not scope by SPIFFE ID")
	assert.Equal(t, "system:serviceaccount:signet:signet-admin", actor,
		"actor must carry the admin's resolved identity for audit attribution (bytepunx/signet#25)")
}

func TestAuthorizeSyncBundle_InvalidBearerTokenFailsClosed(t *testing.T) {
	srv := &GitOpsServer{validator: &fakeTokenValidator{err: auth.ErrInvalidToken}}
	_, _, err := srv.authorizeSyncBundle(bearerCtx("bad-token"))
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthorizeSyncBundle_ValidSPIFFEIdentityScopesAccess(t *testing.T) {
	srv := &GitOpsServer{validator: &fakeTokenValidator{}} // never consulted: no bearer token present
	ctx := spiffeCtx("spiffe://cluster.local/ns/authstar/sa/tower")
	actor, scoped, err := srv.authorizeSyncBundle(ctx)
	require.NoError(t, err)
	assert.True(t, scoped)
	assert.Equal(t, "spiffe://cluster.local/ns/authstar/sa/tower", actor)
}

func TestAuthorizeSyncBundle_NoCredentialsRejected(t *testing.T) {
	srv := &GitOpsServer{validator: &fakeTokenValidator{}}
	_, _, err := srv.authorizeSyncBundle(context.Background())
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

// --- validateBundleScope ---

func writeYAML(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("k: v\n"), 0o600))
}

// TestValidateBundleScope_OwnNamespacePasses is the core regression test for
// bytepunx/signet#23: a workload pushing a secret under its own
// namespace/service must be allowed through, exactly mirroring how reads
// already auto-permit a workload's own namespace/service.
func TestValidateBundleScope_OwnNamespacePasses(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, filepath.Join(dir, "secrets", "authstar", "tower", "tenant-key.yaml"))

	var gotPermission, gotNS, gotSvc, gotName string
	srv := &GitOpsServer{checker: &scopeCheckerFunc{allow: func(_ context.Context, spiffeID, permission, namespace, service, secretName string) error {
		gotPermission, gotNS, gotSvc, gotName = permission, namespace, service, secretName
		if namespace == "authstar" && service == "tower" {
			return nil
		}
		return auth.ErrUnauthorized
	}}}

	err := srv.validateBundleScope(context.Background(), dir, "secrets/", "", "spiffe://cluster.local/ns/authstar/sa/tower")
	require.NoError(t, err)
	assert.Equal(t, "put", gotPermission)
	assert.Equal(t, "authstar", gotNS)
	assert.Equal(t, "tower", gotSvc)
	assert.Equal(t, "tenant-key", gotName)
}

// TestValidateBundleScope_CrossNamespaceRejected verifies a bundle containing
// even one out-of-scope file is rejected outright — the whole point of
// validating before SyncFromDir/SyncConfigFromDir write anything.
func TestValidateBundleScope_CrossNamespaceRejected(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, filepath.Join(dir, "secrets", "authstar", "tower", "own-secret.yaml"))
	writeYAML(t, filepath.Join(dir, "secrets", "payments", "api", "stripe-key.yaml"))

	srv := &GitOpsServer{checker: &scopeCheckerFunc{allow: func(_ context.Context, _, _, namespace, service, _ string) error {
		if namespace == "authstar" && service == "tower" {
			return nil
		}
		return auth.ErrUnauthorized
	}}}

	err := srv.validateBundleScope(context.Background(), dir, "secrets/", "", "spiffe://cluster.local/ns/authstar/sa/tower")
	require.Error(t, err, "a bundle containing an out-of-scope file must be rejected")
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

// TestValidateBundleScope_MalformedPathIgnored verifies scope validation
// doesn't reject files that don't match the path-depth convention — that's
// the syncer's own concern (bytepunx/signet#22), not an authorization
// question; a file scope validation can't identify a namespace/service for
// can't be scope-checked at all.
func TestValidateBundleScope_MalformedPathIgnored(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, filepath.Join(dir, "secrets", "authstar", "too", "deep", "nested.yaml"))

	called := false
	srv := &GitOpsServer{checker: &scopeCheckerFunc{allow: func(context.Context, string, string, string, string, string) error {
		called = true
		return nil
	}}}

	err := srv.validateBundleScope(context.Background(), dir, "secrets/", "", "spiffe://cluster.local/ns/authstar/sa/tower")
	require.NoError(t, err)
	assert.False(t, called, "a file the parser can't identify a namespace/service for must not reach the checker")
}

// TestValidateBundleScope_ConfigPathUsesEmptySecretName verifies config
// files (which have no third path segment) are checked with an empty
// secretName, matching the existing bundle/service-level Allow convention
// used elsewhere (e.g. GetServiceBundle).
func TestValidateBundleScope_ConfigPathUsesEmptySecretName(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, filepath.Join(dir, "config", "authstar", "tower.yaml"))

	var gotName string
	nameSeen := false
	srv := &GitOpsServer{checker: &scopeCheckerFunc{allow: func(_ context.Context, _, _, _, _, secretName string) error {
		gotName, nameSeen = secretName, true
		return nil
	}}}

	err := srv.validateBundleScope(context.Background(), dir, "", "config/", "spiffe://cluster.local/ns/authstar/sa/tower")
	require.NoError(t, err)
	require.True(t, nameSeen)
	assert.Empty(t, gotName)
}
