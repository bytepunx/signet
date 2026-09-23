package gitops

import (
	"context"
	"log/slog"
	"time"
)

// Reconciler periodically performs a full sync of all registered repositories
// to catch events missed during downtime.
type Reconciler struct {
	store    secretStore
	syncer   *Syncer
	interval time.Duration
}

// DefaultReconcileInterval is used when no interval is specified.
const DefaultReconcileInterval = 5 * time.Minute

// NewReconciler constructs a Reconciler. interval <= 0 uses DefaultReconcileInterval.
func NewReconciler(st secretStore, syncer *Syncer, interval time.Duration) *Reconciler {
	if interval <= 0 {
		interval = DefaultReconcileInterval
	}
	return &Reconciler{store: st, syncer: syncer, interval: interval}
}

// Run performs an immediate full reconciliation and then repeats at the
// configured interval until ctx is cancelled.
func (r *Reconciler) Run(ctx context.Context) error {
	r.reconcileAll(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			r.reconcileAll(ctx)
		}
	}
}

func (r *Reconciler) reconcileAll(ctx context.Context) {
	repos, err := r.store.ListRepositories(ctx)
	if err != nil {
		slog.Error("reconcile: list repositories", "err", err)
		return
	}
	for i := range repos {
		repo := &repos[i]

		// Skip the full clone-and-walk entirely when the remote hasn't moved
		// since the last sync — otherwise every tick re-processes every
		// secret and (pre-bytepunx/signet#55 fix) falsely reports them all
		// as "added". This is a pure optimization: ResolveHeadSHA failing
		// just means we fall through to the unconditional FullSync below,
		// same as before this check existed.
		if repo.LastSyncSHA != nil {
			headSHA, err := r.syncer.ResolveHeadSHA(ctx, repo)
			if err != nil {
				slog.Warn("reconcile: resolve head sha, falling back to full sync", "repo", repo.Name, "err", err)
			} else if headSHA == *repo.LastSyncSHA {
				slog.Debug("reconcile: no change since last sync, skipping", "repo", repo.Name, "sha", headSHA)
				continue
			}
		}

		result, err := r.syncer.FullSync(ctx, repo, "repo:"+repo.Name, false)
		if err != nil {
			slog.Error("reconcile: sync failed", "repo", repo.Name, "err", err)
			continue
		}
		slog.Info("reconcile: sync complete",
			"repo", repo.Name,
			"sha", result.SHA,
			"added", result.Added,
			"deleted", result.Deleted,
		)
	}
}
