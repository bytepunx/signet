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
		result, err := r.syncer.FullSync(ctx, repo)
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
