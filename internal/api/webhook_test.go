package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/gitops"
	"github.com/bytepunx/signet/internal/store"
	"github.com/bytepunx/signet/internal/unseal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeWebhookStore implements webhookStore for testing.
type fakeWebhookStore struct {
	repo    *store.Repository
	repoErr error
}

func (f *fakeWebhookStore) GetRepositoryByName(_ context.Context, _ string) (*store.Repository, error) {
	return f.repo, f.repoErr
}
func (f *fakeWebhookStore) GetRepository(_ context.Context, _ string) (*store.Repository, error) {
	return f.repo, f.repoErr
}

// stubGitopsStore implements the unexported gitops.secretStore interface with
// no-ops; the webhook test paths below never reach syncer.SyncFromPush, so
// none of these are actually called, but gitops.NewSyncer requires a value.
type stubGitopsStore struct{}

func (stubGitopsStore) PutSecret(_ context.Context, _ *store.Secret) error { return nil }
func (stubGitopsStore) GetSecret(_ context.Context, _, _, _ string) (*store.Secret, error) {
	return nil, store.ErrNotFound
}
func (stubGitopsStore) DeleteSecret(_ context.Context, _, _, _ string) error { return nil }
func (stubGitopsStore) ListSOPSKeys(_ context.Context, _ string) ([]store.SOPSKey, error) {
	return nil, nil
}
func (stubGitopsStore) GetRepository(_ context.Context, _ string) (*store.Repository, error) {
	return nil, store.ErrNotFound
}
func (stubGitopsStore) ListRepositories(_ context.Context) ([]store.Repository, error) {
	return nil, nil
}
func (stubGitopsStore) UpdateSyncState(_ context.Context, _, _ string, _ time.Time) error {
	return nil
}
func (stubGitopsStore) PutServiceConfig(_ context.Context, _, _ string, _ json.RawMessage, _ string) error {
	return nil
}
func (stubGitopsStore) DeleteServiceConfig(_ context.Context, _, _ string) error { return nil }
func (stubGitopsStore) GetActiveKEK(_ context.Context) (*store.KEK, error) {
	return nil, store.ErrNotFound
}
func (stubGitopsStore) PutKEK(_ context.Context, _ *store.KEK) error { return nil }
func (stubGitopsStore) ListSecretKeysForRepo(_ context.Context, _ string) ([]store.SecretKey, error) {
	return nil, nil
}
func (stubGitopsStore) ListConfigKeysForRepo(_ context.Context, _ string) ([]store.ConfigKey, error) {
	return nil, nil
}

func newTestWebhookHandler(t *testing.T, repo *store.Repository, sealed bool) *WebhookHandler {
	t.Helper()
	state := unseal.StateUnsealed
	if sealed {
		state = unseal.StateSealed
	}
	syncer := gitops.NewSyncer(stubGitopsStore{}, &fakeKeyUnwrapper{}, nil, "")
	return NewWebhookHandler(
		&fakeWebhookStore{repo: repo},
		&fakeKeyUnwrapper{key: masterKeyForWebhookTests},
		syncer,
		&fakeUnsealMgr{statusResult: unseal.Status{State: state}},
	)
}

var masterKeyForWebhookTests = make([]byte, icrypto.KeySize)

// buildWebhookRepo creates a store.Repository with a real encrypted webhook
// secret under masterKeyForWebhookTests, bound via AAD to the repo name, and
// returns the repo plus the plaintext hex secret used to sign requests.
func buildWebhookRepo(t *testing.T, name, branch string) (*store.Repository, string) {
	t.Helper()
	secretHex := "deadbeef00112233445566778899aabbccddeeff0011223344556677889900"
	ct, err := icrypto.Encrypt(masterKeyForWebhookTests, []byte(secretHex), icrypto.BindAAD(icrypto.AADRepoWebhookSecret, name))
	require.NoError(t, err)
	return &store.Repository{
		ID: "repo-1", Name: name, Branch: branch,
		EncryptedWebhookSecret: ct,
		EncryptedDeployKey:     []byte("unused"),
	}, secretHex
}

// sign computes the webhook HMAC using secretHex's own ASCII bytes as the
// key — matching production, where the hex string itself (not its decoded
// bytes) is both what's stored/decrypted and what an operator pastes into
// GitHub's webhook secret field.
func sign(t *testing.T, secretHex string, body []byte) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secretHex))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestWebhook_SealedReturns503(t *testing.T) {
	repo, secretHex := buildWebhookRepo(t, "infra", "main")
	h := newTestWebhookHandler(t, repo, true)

	body := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook/github/repo-1", strings.NewReader(string(body)))
	req.Header.Set("X-Hub-Signature-256", sign(t, secretHex, body))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestWebhook_WrongMethodReturns405(t *testing.T) {
	repo, _ := buildWebhookRepo(t, "infra", "main")
	h := newTestWebhookHandler(t, repo, false)

	req := httptest.NewRequest(http.MethodGet, "/webhook/github/repo-1", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestWebhook_MalformedRepoIDReturns404(t *testing.T) {
	repo, _ := buildWebhookRepo(t, "infra", "main")
	h := newTestWebhookHandler(t, repo, false)

	req := httptest.NewRequest(http.MethodPost, "/webhook/github/repo-1/extra", strings.NewReader("{}"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestWebhook_UnknownRepoAndBadSignature_IndistinguishableResponse is the M-6
// enumeration regression test: an unknown repo_id and a known repo with a bad
// signature must return the identical status and body.
func TestWebhook_UnknownRepoAndBadSignature_IndistinguishableResponse(t *testing.T) {
	body := []byte(`{}`)

	unknownRepoHandler := NewWebhookHandler(
		&fakeWebhookStore{repoErr: store.ErrNotFound},
		&fakeKeyUnwrapper{key: masterKeyForWebhookTests},
		gitops.NewSyncer(stubGitopsStore{}, &fakeKeyUnwrapper{}, nil, ""),
		&fakeUnsealMgr{statusResult: unseal.Status{State: unseal.StateUnsealed}},
	)
	req1 := httptest.NewRequest(http.MethodPost, "/webhook/github/unknown-repo", strings.NewReader(string(body)))
	req1.Header.Set("X-Hub-Signature-256", "sha256=0000000000000000000000000000000000000000000000000000000000000000")
	rec1 := httptest.NewRecorder()
	unknownRepoHandler.ServeHTTP(rec1, req1)

	repo, _ := buildWebhookRepo(t, "infra", "main")
	badSigHandler := newTestWebhookHandler(t, repo, false)
	req2 := httptest.NewRequest(http.MethodPost, "/webhook/github/repo-1", strings.NewReader(string(body)))
	req2.Header.Set("X-Hub-Signature-256", "sha256=0000000000000000000000000000000000000000000000000000000000000000")
	rec2 := httptest.NewRecorder()
	badSigHandler.ServeHTTP(rec2, req2)

	require.Equal(t, http.StatusUnauthorized, rec1.Code)
	require.Equal(t, http.StatusUnauthorized, rec2.Code)
	assert.Equal(t, rec1.Code, rec2.Code)
	assert.Equal(t, strings.TrimSpace(rec1.Body.String()), strings.TrimSpace(rec2.Body.String()))
}

func TestWebhook_NonPushEventAcceptedButIgnored(t *testing.T) {
	repo, secretHex := buildWebhookRepo(t, "infra", "main")
	h := newTestWebhookHandler(t, repo, false)

	body := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook/github/repo-1", strings.NewReader(string(body)))
	req.Header.Set("X-Hub-Signature-256", sign(t, secretHex, body))
	req.Header.Set("X-GitHub-Event", "ping")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestWebhook_RateLimiting is the M-6 regression test: once the burst is
// exhausted, further requests are rejected with 429 rather than being
// processed (each of which would otherwise force a master-key decrypt).
func TestWebhook_RateLimiting(t *testing.T) {
	repo, _ := buildWebhookRepo(t, "infra", "main")
	h := newTestWebhookHandler(t, repo, false)

	// Use a cheap path (wrong method -> 405) to isolate the rate limiter's
	// own behavior from the rest of the handler.
	var lastCode int
	for i := 0; i < webhookRateBurst+1; i++ {
		req := httptest.NewRequest(http.MethodGet, "/webhook/github/repo-1", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		lastCode = rec.Code
	}
	assert.Equal(t, http.StatusTooManyRequests, lastCode,
		"the request beyond the configured burst must be rate-limited")
}
