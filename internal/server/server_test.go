package server_test

import (
	"context"
	"sync"
	"testing"
	"time"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	signetv1 "github.com/bytepunx/signet/gen/signet/v1"
	"github.com/bytepunx/signet/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// --- minimal stubs ---

// panicServer exercises the panic recovery interceptor.
type panicSecretsServer struct{ signetv1.UnimplementedSecretsServiceServer }

func (panicSecretsServer) GetSecret(context.Context, *signetv1.GetSecretRequest) (*signetv1.GetSecretResponse, error) {
	panic("deliberate panic in handler")
}

type stubAdmin struct{ adminv1.UnimplementedAdminServiceServer }
type stubGitOps struct{ adminv1.UnimplementedGitOpsServiceServer }
type stubSecrets struct{ signetv1.UnimplementedSecretsServiceServer }

type fakeMgr struct {
	mu     sync.Mutex
	sealed bool
}

func (f *fakeMgr) Seal() {
	f.mu.Lock()
	f.sealed = true
	f.mu.Unlock()
}

func (f *fakeMgr) isSealed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sealed
}

// newTestServer creates a Server listening on random loopback ports using
// insecure credentials (no SPIRE required in tests).
func newTestServer(t *testing.T, secrets signetv1.SecretsServiceServer, admin adminv1.AdminServiceServer, gitops adminv1.GitOpsServiceServer, mgr *fakeMgr) *server.Server {
	t.Helper()
	if secrets == nil {
		secrets = stubSecrets{}
	}
	if admin == nil {
		admin = stubAdmin{}
	}
	if gitops == nil {
		gitops = stubGitOps{}
	}
	srv, err := server.New(
		server.Config{WorkloadAddr: "127.0.0.1:0", AdminAddr: "127.0.0.1:0"},
		insecure.NewCredentials(),
		secrets, admin, gitops, nil, mgr,
	)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	return srv
}

// runBackground starts srv.Run in a goroutine and returns a cancel func plus
// a done channel that closes when Run returns.
func runBackground(srv *server.Server) (cancel context.CancelFunc, done <-chan error) {
	ctx, c := context.WithCancel(context.Background())
	ch := make(chan error, 1)
	go func() { ch <- srv.Run(ctx) }()
	return c, ch
}

// waitReady sleeps briefly to let both listeners accept connections.
func waitReady() { time.Sleep(30 * time.Millisecond) }

// --- New validation ---

func TestNew_MissingWorkloadAddr(t *testing.T) {
	_, err := server.New(server.Config{AdminAddr: ":0"}, insecure.NewCredentials(),
		stubSecrets{}, stubAdmin{}, stubGitOps{}, nil, &fakeMgr{})
	if err == nil {
		t.Fatal("expected error for missing WorkloadAddr")
	}
}

func TestNew_MissingAdminAddr(t *testing.T) {
	_, err := server.New(server.Config{WorkloadAddr: ":0"}, insecure.NewCredentials(),
		stubSecrets{}, stubAdmin{}, stubGitOps{}, nil, &fakeMgr{})
	if err == nil {
		t.Fatal("expected error for missing AdminAddr")
	}
}

func TestNew_NilCredentials(t *testing.T) {
	_, err := server.New(server.Config{WorkloadAddr: ":0", AdminAddr: ":0"}, nil,
		stubSecrets{}, stubAdmin{}, stubGitOps{}, nil, &fakeMgr{})
	if err == nil {
		t.Fatal("expected error for nil credentials")
	}
}

// --- Lifecycle ---

func TestRun_CleanShutdownReturnsNil(t *testing.T) {
	srv := newTestServer(t, nil, nil, nil, &fakeMgr{})
	cancel, done := runBackground(srv)
	waitReady()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil on clean shutdown, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Run to return")
	}
}

func TestRun_SealCalledOnShutdown(t *testing.T) {
	mgr := &fakeMgr{}
	srv := newTestServer(t, nil, nil, nil, mgr)
	cancel, done := runBackground(srv)
	waitReady()
	cancel()
	<-done
	if !mgr.isSealed() {
		t.Error("expected Seal to be called on shutdown")
	}
}

func TestRun_DefaultDrainTimeout(t *testing.T) {
	// DrainTimeout: 0 should not panic and should default gracefully.
	srv, err := server.New(
		server.Config{WorkloadAddr: "127.0.0.1:0", AdminAddr: "127.0.0.1:0", DrainTimeout: 0},
		insecure.NewCredentials(),
		stubSecrets{}, stubAdmin{}, stubGitOps{}, nil, &fakeMgr{},
	)
	if err != nil {
		t.Fatal(err)
	}
	cancel, done := runBackground(srv)
	waitReady()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// --- Reachability ---

func TestRun_WorkloadListenerReachable(t *testing.T) {
	srv := newTestServer(t, nil, nil, nil, &fakeMgr{})
	cancel, _ := runBackground(srv)
	defer cancel()
	waitReady()

	conn, err := grpc.NewClient(srv.WorkloadAddr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial workload: %v", err)
	}
	defer conn.Close()

	// Unimplemented stub returns Unimplemented; any gRPC response confirms connectivity.
	_, err = signetv1.NewSecretsServiceClient(conn).GetSecret(context.Background(), &signetv1.GetSecretRequest{
		Namespace: "ns", Service: "svc", Name: "k",
	})
	if status.Code(err) != codes.Unimplemented {
		t.Errorf("want Unimplemented, got %v", err)
	}
}

func TestRun_AdminListenerReachable(t *testing.T) {
	srv := newTestServer(t, nil, nil, nil, &fakeMgr{})
	cancel, _ := runBackground(srv)
	defer cancel()
	waitReady()

	conn, err := grpc.NewClient(srv.AdminAddr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial admin: %v", err)
	}
	defer conn.Close()

	_, err = adminv1.NewAdminServiceClient(conn).Status(context.Background(), &adminv1.StatusRequest{})
	if status.Code(err) != codes.Unimplemented {
		t.Errorf("want Unimplemented, got %v", err)
	}
}

func TestRun_GitOpsListenerReachable(t *testing.T) {
	srv := newTestServer(t, nil, nil, nil, &fakeMgr{})
	cancel, _ := runBackground(srv)
	defer cancel()
	waitReady()

	conn, err := grpc.NewClient(srv.AdminAddr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial admin: %v", err)
	}
	defer conn.Close()

	_, err = adminv1.NewGitOpsServiceClient(conn).GetSOPSPublicKey(context.Background(), &adminv1.GetSOPSPublicKeyRequest{})
	if status.Code(err) != codes.Unimplemented {
		t.Errorf("want Unimplemented, got %v", err)
	}
}

// --- Panic recovery ---

func TestRun_PanicInHandlerReturnsInternal(t *testing.T) {
	srv := newTestServer(t, panicSecretsServer{}, nil, nil, &fakeMgr{})
	cancel, _ := runBackground(srv)
	defer cancel()
	waitReady()

	conn, err := grpc.NewClient(srv.WorkloadAddr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// First call: panic is caught, returns Internal.
	_, err = signetv1.NewSecretsServiceClient(conn).GetSecret(context.Background(), &signetv1.GetSecretRequest{
		Namespace: "ns", Service: "svc", Name: "k",
	})
	if status.Code(err) != codes.Internal {
		t.Errorf("want Internal after panic recovery, got %v", err)
	}

	// Second call on the same connection confirms the server is still alive.
	_, err = signetv1.NewSecretsServiceClient(conn).GetSecret(context.Background(), &signetv1.GetSecretRequest{
		Namespace: "ns", Service: "svc", Name: "k",
	})
	if status.Code(err) != codes.Internal {
		t.Errorf("server should still respond after panic recovery, got %v", err)
	}
}

// --- Addr helpers ---

func TestServer_AddrHelpers(t *testing.T) {
	srv := newTestServer(t, nil, nil, nil, &fakeMgr{})
	if srv.WorkloadAddr() == nil {
		t.Error("WorkloadAddr() should not be nil")
	}
	if srv.AdminAddr() == nil {
		t.Error("AdminAddr() should not be nil")
	}
	if srv.WorkloadAddr().String() == srv.AdminAddr().String() {
		t.Error("WorkloadAddr and AdminAddr should differ")
	}
}
