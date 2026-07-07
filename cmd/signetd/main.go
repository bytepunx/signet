package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bytepunx/signet/internal/api"
	"github.com/bytepunx/signet/internal/audit"
	"github.com/bytepunx/signet/internal/auth"
	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/gitops"
	"github.com/bytepunx/signet/internal/server"
	"github.com/bytepunx/signet/internal/store"
	"github.com/bytepunx/signet/internal/unseal"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "signetd: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Cancel on SIGTERM or SIGINT; stop clears the signal registration so a
	// second signal terminates the process immediately.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// --- build dependencies in order ---

	// Database (runs migrations on first connection).
	st, err := store.New(ctx, cfg.DBConnString)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	defer st.Close()

	// Crypto key store — holds the in-memory master key behind memguard.
	keyStore := icrypto.NewKeyStore()

	// Unseal manager — drives the sealed → unsealed state machine.
	unsealMgr, err := unseal.New(keyStore, unseal.Config{
		ShamirShares:    cfg.ShamirShares,
		ShamirThreshold: cfg.ShamirThreshold,
		ShareTimeout:    cfg.ShareTimeout,
	})
	if err != nil {
		return fmt.Errorf("unseal manager: %w", err)
	}

	// Audit writer — HMAC-SHA256 chain; chain key is zeroed on shutdown.
	auditWriter, err := audit.New(ctx, st, cfg.auditKeyBytes)
	if err != nil {
		return fmt.Errorf("audit writer: %w", err)
	}
	defer auditWriter.Zero()

	// Policy checker — evaluates glob-matched namespace/secret policies.
	checker := auth.NewChecker(st)

	// Kubernetes client for SA token validation on the admin endpoint.
	k8sClient, err := buildK8sClient()
	if err != nil {
		return fmt.Errorf("kubernetes client: %w", err)
	}
	tokenValidator := auth.NewTokenValidator(k8sClient, cfg.kubeAudiences, cfg.adminSubjects)

	// API handlers.
	bus := api.NewBus()
	lockMgr := api.NewLockManager(st)
	go lockMgr.Run(ctx)
	secretsSrv := api.NewSecretsServer(st, keyStore, checker, auditWriter, bus, lockMgr, cfg.AuditFailClosed)
	adminSrv := api.NewAdminServer(unsealMgr, tokenValidator, st, keyStore)

	// GitOps layer.
	syncer := gitops.NewSyncer(st, keyStore, bus, cfg.Environment)
	reconciler := gitops.NewReconciler(st, syncer, 0)
	gitopsSrv := api.NewGitOpsServer(st, keyStore, syncer, cfg.WebhookBaseURL, tokenValidator, cfg.Environment)
	webhookHandler := api.NewWebhookHandler(st, keyStore, syncer, unsealMgr)

	// Kubernetes auto-unseal: if configured, fetch the master key from a Secret
	// and unseal before the server starts accepting traffic. Runs synchronously
	// so any startup-probe delay occurs before listeners are bound, not after.
	// Failure is non-fatal — server starts sealed and manual unseal still applies.
	if cfg.KubeUnsealSecret != "" {
		slog.Warn("kube-unseal enabled: the master key will be stored in a Kubernetes Secret as "+
			"plaintext base64. This requires etcd encryption-at-rest to be configured on the "+
			"cluster (https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/) — signetd "+
			"cannot verify this itself. Without it, the master key is recoverable from etcd/backups "+
			"independent of Kubernetes RBAC.",
			"secret", cfg.KubeUnsealSecret)
		unsealCtx, unsealCancel := context.WithTimeout(ctx, 30*time.Second)
		attemptKubeUnseal(unsealCtx, unsealMgr, st, keyStore, cfg.KubeUnsealSecret)
		unsealCancel()
	}

	// Drive the reconciler lifecycle from the unseal state machine.
	// The reconciler only runs while the server is unsealed — it cannot decrypt
	// stored age keys when sealed. On seal it is cancelled; on unseal it restarts.
	go runReconcilerLifecycle(ctx, unsealMgr, reconciler)

	// gRPC server — dials SPIRE for mTLS credentials, binds both listeners.
	srv, err := server.NewFromSPIRE(
		ctx,
		server.Config{
			WorkloadAddr: cfg.WorkloadAddr,
			AdminAddr:    cfg.AdminAddr,
			WebhookAddr:  cfg.WebhookAddr,
			DrainTimeout: cfg.DrainTimeout,
		},
		cfg.SpireSocket,
		cfg.TrustDomain,
		secretsSrv,
		adminSrv,
		gitopsSrv,
		webhookHandler,
		unsealMgr,
	)
	if err != nil {
		return fmt.Errorf("grpc server: %w", err)
	}

	logStartup(cfg)

	if err := srv.Run(ctx); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	slog.Info("signetd stopped cleanly")
	return nil
}

// buildK8sClient returns a Kubernetes client using the in-cluster service
// account token and CA cert. signetd must run inside a Kubernetes pod with
// a mounted service account that has permission to create TokenReview objects.
func buildK8sClient() (kubernetes.Interface, error) {
	rcfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config: %w (is signetd running inside Kubernetes?)", err)
	}
	client, err := kubernetes.NewForConfig(rcfg)
	if err != nil {
		return nil, fmt.Errorf("kubernetes client: %w", err)
	}
	return client, nil
}

// runReconcilerLifecycle watches the unseal state channel and manages the
// reconciler goroutine: starting it on unseal and cancelling it on seal.
// Returns when ctx is cancelled (server shutdown).
func runReconcilerLifecycle(ctx context.Context, mgr *unseal.Manager, rec *gitops.Reconciler) {
	var cancelReconciler context.CancelFunc

	start := func() {
		if cancelReconciler != nil {
			return // already running
		}
		rctx, cancel := context.WithCancel(ctx)
		cancelReconciler = cancel
		go func() {
			if err := rec.Run(rctx); err != nil && !errors.Is(err, context.Canceled) {
				slog.Error("reconciler exited unexpectedly", "err", err)
			}
		}()
		slog.Info("gitops reconciler started")
	}

	stop := func() {
		if cancelReconciler != nil {
			cancelReconciler()
			cancelReconciler = nil
			slog.Info("gitops reconciler stopped")
		}
	}
	defer stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mgr.StatusCh():
			if mgr.Status().State == unseal.StateUnsealed {
				start()
			} else {
				stop()
			}
		}
	}
}

// logStartup emits a structured startup log. It never logs secrets, keys, or
// connection strings — only configuration shape and listener addresses.
func logStartup(cfg config) {
	shamirMode := cfg.ShamirThreshold > 0
	attrs := []any{
		"workload_addr", cfg.WorkloadAddr,
		"admin_addr", cfg.AdminAddr,
		"trust_domain", cfg.TrustDomain,
		"state", "sealed",
		"shamir_mode", shamirMode,
	}
	if shamirMode {
		attrs = append(attrs,
			"shamir_threshold", cfg.ShamirThreshold,
			"shamir_shares", cfg.ShamirShares,
		)
	}
	slog.Info("signetd starting", attrs...)
}
