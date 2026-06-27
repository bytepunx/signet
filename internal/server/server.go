// Package server wires together all dependencies and manages the server lifecycle.
package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	signetv1 "github.com/bytepunx/signet/gen/signet/v1"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	grpccredentials "github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

// Config holds the addresses and timing parameters for a Server.
type Config struct {
	// WorkloadAddr is the TCP address for the mTLS gRPC workload listener (e.g. ":8443").
	WorkloadAddr string
	// AdminAddr is the TCP address for the admin gRPC listener (e.g. "127.0.0.1:8444").
	// This listener has no TLS; it is exposed only via kubectl port-forward in production.
	AdminAddr string
	// WebhookAddr is the TCP address for the optional GitHub webhook HTTP listener
	// (e.g. ":8445"). Leave empty to disable the webhook server.
	WebhookAddr string
	// DrainTimeout is how long Run waits for in-flight RPCs to complete before
	// force-stopping both servers. Defaults to 30s if zero.
	DrainTimeout time.Duration
}

type sealable interface {
	Seal()
}

// Server manages two gRPC listeners and an optional HTTP webhook listener:
//   - workload  — mTLS, serves SecretsService to workloads identified by SPIFFE SVIDs.
//   - admin     — plain TCP, serves AdminService and GitOpsService to operators via port-forward.
//   - webhook   — HTTP, serves GitHub push webhooks (optional, enabled by Config.WebhookAddr).
type Server struct {
	cfg         Config
	workloadSrv *grpc.Server
	adminSrv    *grpc.Server
	webhookSrv  *http.Server  // nil when WebhookAddr is empty
	workloadLis net.Listener
	adminLis    net.Listener
	webhookLis  net.Listener  // nil when WebhookAddr is empty
	mgr         sealable
	closer      io.Closer // X509Source; nil in tests using insecure credentials
}

// New wires the gRPC servers and starts listening on both addresses.
// webhook may be nil; if provided and cfg.WebhookAddr is non-empty, an HTTP
// server is also started for GitHub push events.
//
// creds is applied to the workload server. In production this should be
// grpccredentials.MTLSServerCredentials(...) from SpireCredentials. In tests,
// pass google.golang.org/grpc/credentials/insecure.NewCredentials().
func New(
	cfg Config,
	creds credentials.TransportCredentials,
	secrets signetv1.SecretsServiceServer,
	admin adminv1.AdminServiceServer,
	gitops adminv1.GitOpsServiceServer,
	webhook http.Handler,
	mgr sealable,
) (*Server, error) {
	return newWithCloser(cfg, creds, secrets, admin, gitops, webhook, mgr, nil)
}

func newWithCloser(
	cfg Config,
	creds credentials.TransportCredentials,
	secrets signetv1.SecretsServiceServer,
	admin adminv1.AdminServiceServer,
	gitops adminv1.GitOpsServiceServer,
	webhook http.Handler,
	mgr sealable,
	closer io.Closer,
) (*Server, error) {
	if cfg.WorkloadAddr == "" {
		return nil, fmt.Errorf("server: WorkloadAddr is required")
	}
	if cfg.AdminAddr == "" {
		return nil, fmt.Errorf("server: AdminAddr is required")
	}
	if creds == nil {
		return nil, fmt.Errorf("server: workload transport credentials are required")
	}
	if cfg.DrainTimeout <= 0 {
		cfg.DrainTimeout = 30 * time.Second
	}

	workloadLis, err := net.Listen("tcp", cfg.WorkloadAddr)
	if err != nil {
		return nil, fmt.Errorf("server: listen workload %s: %w", cfg.WorkloadAddr, err)
	}

	adminLis, err := net.Listen("tcp", cfg.AdminAddr)
	if err != nil {
		workloadLis.Close()
		return nil, fmt.Errorf("server: listen admin %s: %w", cfg.AdminAddr, err)
	}

	workloadSrv := grpc.NewServer(
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(recoveryInterceptor),
		grpc.ChainStreamInterceptor(recoveryStreamInterceptor),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Time:              30 * time.Second,
			Timeout:           10 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: false,
		}),
	)
	signetv1.RegisterSecretsServiceServer(workloadSrv, secrets)

	adminSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(recoveryInterceptor),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    30 * time.Second,
			Timeout: 10 * time.Second,
		}),
	)
	adminv1.RegisterAdminServiceServer(adminSrv, admin)
	adminv1.RegisterGitOpsServiceServer(adminSrv, gitops)

	// Optional webhook HTTP server.
	var webhookSrv *http.Server
	var webhookLis net.Listener
	if cfg.WebhookAddr != "" && webhook != nil {
		lis, lisErr := net.Listen("tcp", cfg.WebhookAddr)
		if lisErr != nil {
			workloadLis.Close()
			adminLis.Close()
			return nil, fmt.Errorf("server: listen webhook %s: %w", cfg.WebhookAddr, lisErr)
		}
		webhookLis = lis
		webhookSrv = &http.Server{
			Handler:      webhook,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
	}

	return &Server{
		cfg:         cfg,
		workloadSrv: workloadSrv,
		adminSrv:    adminSrv,
		webhookSrv:  webhookSrv,
		workloadLis: workloadLis,
		adminLis:    adminLis,
		webhookLis:  webhookLis,
		mgr:         mgr,
		closer:      closer,
	}, nil
}

// WorkloadAddr returns the address the workload listener is bound to.
// Useful in tests when ":0" is used to let the OS pick a port.
func (s *Server) WorkloadAddr() net.Addr { return s.workloadLis.Addr() }

// AdminAddr returns the address the admin listener is bound to.
func (s *Server) AdminAddr() net.Addr { return s.adminLis.Addr() }

// Run starts both gRPC servers and blocks until ctx is cancelled or a server
// fails. On return both servers are fully stopped and the master key is sealed.
//
// A clean context cancellation (SIGTERM handler) returns nil. An unexpected
// server failure returns a wrapped error identifying which listener failed.
func (s *Server) Run(ctx context.Context) error {
	type result struct {
		name string
		err  error
	}
	listeners := 2
	if s.webhookSrv != nil {
		listeners = 3
	}
	errs := make(chan result, listeners)

	go func() {
		if err := s.workloadSrv.Serve(s.workloadLis); err != nil {
			errs <- result{"workload", err}
		} else {
			errs <- result{"workload", nil}
		}
	}()
	go func() {
		if err := s.adminSrv.Serve(s.adminLis); err != nil {
			errs <- result{"admin", err}
		} else {
			errs <- result{"admin", nil}
		}
	}()
	if s.webhookSrv != nil {
		go func() {
			if err := s.webhookSrv.Serve(s.webhookLis); err != nil && err != http.ErrServerClosed {
				errs <- result{"webhook", err}
			} else {
				errs <- result{"webhook", nil}
			}
		}()
	}

	var firstErr error
	select {
	case <-ctx.Done():
		// Requested shutdown — not an error.
	case r := <-errs:
		if r.err != nil {
			firstErr = fmt.Errorf("server: %s listener: %w", r.name, r.err)
		}
	}

	s.drain()
	// Drain the remaining goroutine results.
	for i := 1; i < listeners; i++ {
		<-errs
	}

	return firstErr
}

// drain gracefully stops all servers in parallel, then seals the master key.
// If the drain period expires before in-flight requests complete, it force-stops.
func (s *Server) drain() {
	done := make(chan struct{})
	go func() {
		var wg sync.WaitGroup
		n := 2
		if s.webhookSrv != nil {
			n = 3
		}
		wg.Add(n)
		go func() { defer wg.Done(); s.workloadSrv.GracefulStop() }()
		go func() { defer wg.Done(); s.adminSrv.GracefulStop() }()
		if s.webhookSrv != nil {
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), s.cfg.DrainTimeout)
				defer cancel()
				_ = s.webhookSrv.Shutdown(ctx)
			}()
		}
		wg.Wait()
		close(done)
	}()

	timer := time.NewTimer(s.cfg.DrainTimeout)
	defer timer.Stop()

	select {
	case <-done:
	case <-timer.C:
		slog.Warn("graceful shutdown timed out; forcing stop", "drain_timeout", s.cfg.DrainTimeout)
		s.workloadSrv.Stop()
		s.adminSrv.Stop()
		if s.webhookSrv != nil {
			_ = s.webhookSrv.Close()
		}
		<-done
	}

	if s.closer != nil {
		if err := s.closer.Close(); err != nil {
			slog.Warn("failed to close X509Source", "err", err)
		}
	}
	s.mgr.Seal()
}

// recoveryInterceptor catches panics in unary handlers and converts them to
// codes.Internal so a single bad request cannot crash the server.
func recoveryInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if p := recover(); p != nil {
			slog.Error("panic in gRPC unary handler", "panic", p)
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}

// recoveryStreamInterceptor catches panics in streaming handlers.
func recoveryStreamInterceptor(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer func() {
		if p := recover(); p != nil {
			slog.Error("panic in gRPC stream handler", "panic", p)
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()
	return handler(srv, ss)
}

// --- SPIRE helpers ---

// SpireSource dials the SPIRE workload API and returns an X509Source that
// provides both the server's SVID and the trust bundle. The caller must close
// the source when the server shuts down (drain() handles this automatically
// when the source is passed to NewFromSPIRE).
func SpireSource(ctx context.Context, socketPath string) (*workloadapi.X509Source, error) {
	source, err := workloadapi.NewX509Source(ctx,
		workloadapi.WithClientOptions(workloadapi.WithAddr(socketPath)),
	)
	if err != nil {
		return nil, fmt.Errorf("server: dial SPIRE at %s: %w", socketPath, err)
	}
	return source, nil
}

// SpireCredentials builds mTLS gRPC transport credentials from a SPIRE
// X509Source. Only SVIDs belonging to trustDomain are accepted by the server.
func SpireCredentials(source *workloadapi.X509Source, trustDomain string) (credentials.TransportCredentials, error) {
	td, err := spiffeid.TrustDomainFromString(trustDomain)
	if err != nil {
		return nil, fmt.Errorf("server: parse trust domain %q: %w", trustDomain, err)
	}
	return grpccredentials.MTLSServerCredentials(source, source, tlsconfig.AuthorizeMemberOf(td)), nil
}

// NewFromSPIRE is a convenience constructor that dials SPIRE, builds mTLS
// credentials, and calls New. The X509Source is closed automatically when the
// server's drain sequence completes. webhook may be nil.
func NewFromSPIRE(
	ctx context.Context,
	cfg Config,
	spireSocket string,
	trustDomain string,
	secrets signetv1.SecretsServiceServer,
	admin adminv1.AdminServiceServer,
	gitops adminv1.GitOpsServiceServer,
	webhook http.Handler,
	mgr sealable,
) (*Server, error) {
	source, err := SpireSource(ctx, spireSocket)
	if err != nil {
		return nil, err
	}
	creds, err := SpireCredentials(source, trustDomain)
	if err != nil {
		source.Close()
		return nil, err
	}
	return newWithCloser(cfg, creds, secrets, admin, gitops, webhook, mgr, source)
}
