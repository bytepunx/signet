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
)

const maxWebhookBodyBytes = 10 << 20 // 10 MiB

// webhookStore is the store subset needed by WebhookHandler.
type webhookStore interface {
	GetRepositoryByName(ctx context.Context, name string) (*store.Repository, error)
	GetRepository(ctx context.Context, id string) (*store.Repository, error)
}

// WebhookHandler serves GitHub push webhook events at:
//
//	POST /webhook/github/{repo_id}
type WebhookHandler struct {
	store  webhookStore
	keys   keyUnwrapper
	syncer *gitops.Syncer
	sealer sealChecker
}

// NewWebhookHandler constructs a WebhookHandler.
func NewWebhookHandler(st webhookStore, keys keyUnwrapper, syncer *gitops.Syncer, sealer sealChecker) *WebhookHandler {
	return &WebhookHandler{store: st, keys: keys, syncer: syncer, sealer: sealer}
}

// ServeHTTP routes POST /webhook/github/{repo_id} and rejects everything else.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	// Fetch repository config.
	repo, err := h.store.GetRepository(ctx, repoID)
	if err != nil {
		slog.Warn("webhook: unknown repo", "repo_id", repoID, "err", err)
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// Decrypt the stored webhook secret.
	var webhookSecret []byte
	if decErr := h.keys.Use(func(masterKey []byte) error {
		plain, dErr := icrypto.Decrypt(masterKey, repo.EncryptedWebhookSecret)
		webhookSecret = plain
		return dErr
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
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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
