package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"

	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/gitops"
	"github.com/bytepunx/signet/internal/store"
	"github.com/bytepunx/signet/internal/unseal"
	"golang.org/x/time/rate"
)

const maxWebhookBodyBytes = 10 << 20 // 10 MiB

// Webhook traffic is bursty around deploys but otherwise low-volume; these
// defaults are generous for legitimate GitHub delivery while still bounding
// the crypto/CPU cost an unauthenticated caller can force per second.
const (
	webhookRateLimit = 20 // requests/sec, sustained
	webhookRateBurst = 40
)

// unauthorizedMsg is returned for both an unknown repo_id and a bad HMAC
// signature, so the two cases are not distinguishable from the response
// alone — a caller cannot use the response to enumerate valid repo IDs.
const unauthorizedMsg = "unauthorized"

// webhookStore is the store subset needed by WebhookHandler.
type webhookStore interface {
	GetRepositoryByName(ctx context.Context, name string) (*store.Repository, error)
	GetRepository(ctx context.Context, id string) (*store.Repository, error)
}

// WebhookHandler serves GitHub push webhook events at:
//
//	POST /webhook/github/{repo_id}
type WebhookHandler struct {
	store   webhookStore
	keys    keyUnwrapper
	syncer  *gitops.Syncer
	sealer  sealChecker
	limiter *rate.Limiter
}

// NewWebhookHandler constructs a WebhookHandler.
func NewWebhookHandler(st webhookStore, keys keyUnwrapper, syncer *gitops.Syncer, sealer sealChecker) *WebhookHandler {
	return &WebhookHandler{
		store: st, keys: keys, syncer: syncer, sealer: sealer,
		limiter: rate.NewLimiter(rate.Limit(webhookRateLimit), webhookRateBurst),
	}
}

// ServeHTTP routes POST /webhook/github/{repo_id} and rejects everything else.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Bound how much CPU/crypto work an unauthenticated caller can force per
	// second, regardless of which branch below they hit.
	if !h.limiter.Allow() {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}

	// Reject requests while sealed — GitHub will retry delivery automatically.
	if h.sealer.Status().State != unseal.StateUnsealed {
		w.Header().Set("Retry-After", "30")
		http.Error(w, "service unavailable: server is sealed", http.StatusServiceUnavailable)
		return
	}

	// Only POST is valid for GitHub webhooks.
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract repo_id from /webhook/github/<repo_id>.
	repoID := strings.TrimPrefix(r.URL.Path, "/webhook/github/")
	if repoID == "" || strings.ContainsAny(repoID, "/") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Read and size-limit the body.
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodyBytes))
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()

	// Fetch repository config. An unknown repo_id and a bad signature (below)
	// return the identical response so the endpoint cannot be used to
	// enumerate valid repo IDs; the distinguishing detail is only logged.
	repo, err := h.store.GetRepository(ctx, repoID)
	if err != nil {
		slog.Warn("webhook: unknown repo", "repo_id", repoID, "err", err)
		http.Error(w, unauthorizedMsg, http.StatusUnauthorized)
		return
	}

	// Decrypt the stored webhook secret.
	var webhookSecret []byte
	if decErr := h.keys.Use(func(masterKey []byte) error {
		plain, legacy, dErr := icrypto.DecryptWithFallback(masterKey, repo.EncryptedWebhookSecret, icrypto.BindAAD(icrypto.AADRepoWebhookSecret, repo.Name))
		if dErr != nil {
			return dErr
		}
		if legacy {
			slog.Warn("webhook secret decrypted via legacy unbound fallback; re-register the repo to re-bind it", "repo", repo.Name)
		}
		webhookSecret = plain
		return nil
	}); decErr != nil {
		slog.Error("webhook: decrypt webhook secret", "repo", repo.Name, "err", decErr)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer gitops.ZeroBytes(webhookSecret)

	// Verify HMAC signature.
	sig := r.Header.Get("X-Hub-Signature-256")
	if err := gitops.VerifyWebhookSignature(body, sig, webhookSecret); err != nil {
		slog.Warn("webhook: invalid signature", "repo", repo.Name)
		http.Error(w, unauthorizedMsg, http.StatusUnauthorized)
		return
	}

	// Only act on push events.
	if r.Header.Get("X-GitHub-Event") != "push" {
		w.WriteHeader(http.StatusOK) // accepted but ignored
		return
	}

	// Parse and validate the push event.
	event, err := gitops.ParsePushEvent(body)
	if err != nil {
		slog.Warn("webhook: parse push event", "repo", repo.Name, "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Ignore pushes to branches we don't track.
	branch := gitops.BranchFromRef(event.GetRef())
	if branch != repo.Branch {
		slog.Debug("webhook: ignoring push to non-tracked branch", "branch", branch, "tracked", repo.Branch)
		w.WriteHeader(http.StatusOK)
		return
	}

	headSHA := event.GetAfter()
	changed, deleted := gitops.ChangedFiles(event)

	result, err := h.syncer.SyncFromPush(ctx, repo, headSHA, changed, deleted)
	if err != nil {
		slog.Error("webhook: sync failed", "repo", repo.Name, "err", err)
		http.Error(w, "sync error", http.StatusInternalServerError)
		return
	}

	slog.Info("webhook: sync complete",
		"repo", repo.Name,
		"sha", result.SHA,
		"added", result.Added,
		"deleted", result.Deleted,
	)
	w.WriteHeader(http.StatusOK)
}
