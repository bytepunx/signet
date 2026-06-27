package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"math/big"
	"net/url"
	"sync"
	"testing"
	"time"

	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/audit"
	"github.com/bytepunx/signet/internal/auth"
	"github.com/bytepunx/signet/internal/store"
	signetv1 "github.com/bytepunx/signet/gen/signet/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// --- fakes ---

type fakeSecretFetcher struct {
	mu     sync.Mutex
	secret *store.Secret
	err    error
}

func (f *fakeSecretFetcher) GetSecret(_ context.Context, _, _, _ string) (*store.Secret, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.secret, f.err
}

func (f *fakeSecretFetcher) GetServiceConfig(_ context.Context, _, _ string) (json.RawMessage, int, error) {
	return nil, 0, store.ErrNotFound
}

func (f *fakeSecretFetcher) FetchServiceSecrets(_ context.Context, _, _ string) ([]store.Secret, error) {
	return nil, nil
}

type fakeKeyUnwrapper struct {
	key []byte // nil means return ErrKeyNotSet
	err error  // if non-nil, Use returns this without calling fn
}

func (f *fakeKeyUnwrapper) Use(fn func([]byte) error) error {
	if f.err != nil {
		return f.err
	}
	if f.key == nil {
		return icrypto.ErrKeyNotSet
	}
	return fn(f.key)
}

type fakeChecker struct {
	err error // nil = permitted
}

func (f *fakeChecker) Allow(_ context.Context, _, _, _, _, _ string) error { return f.err }

type fakeRecorder struct {
	mu      sync.Mutex
	entries []audit.Entry
	err     error
}

func (f *fakeRecorder) Record(_ context.Context, e audit.Entry) error {
	f.mu.Lock()
	f.entries = append(f.entries, e)
	f.mu.Unlock()
	return f.err
}

func (f *fakeRecorder) last() audit.Entry {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.entries) == 0 {
		return audit.Entry{}
	}
	return f.entries[len(f.entries)-1]
}

// fakeWatchStream is a fake grpc.ServerStreamingServer[signetv1.WatchSecretResponse].
type fakeWatchStream struct {
	ctx     context.Context
	sends   chan *signetv1.WatchSecretResponse
	sendErr error
}

func newFakeStream(ctx context.Context) *fakeWatchStream {
	return &fakeWatchStream{ctx: ctx, sends: make(chan *signetv1.WatchSecretResponse, 16)}
}

func (f *fakeWatchStream) Send(r *signetv1.WatchSecretResponse) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sends <- r
	return nil
}
func (f *fakeWatchStream) Context() context.Context      { return f.ctx }
func (f *fakeWatchStream) SetHeader(metadata.MD) error   { return nil }
func (f *fakeWatchStream) SendHeader(metadata.MD) error  { return nil }
func (f *fakeWatchStream) SetTrailer(metadata.MD)        {}
func (f *fakeWatchStream) SendMsg(any) error             { return nil }
func (f *fakeWatchStream) RecvMsg(any) error             { return nil }

// fakeLockStore satisfies lockStore with no-op implementations for tests that
// don't exercise the restart lock path.
type fakeLockStore struct{}

func (f *fakeLockStore) TryAcquireLock(_ context.Context, _, _, _ string, _ time.Time) (bool, error) {
	return false, nil
}
func (f *fakeLockStore) HeartbeatLock(_ context.Context, _, _, _ string, _ time.Time) (time.Time, error) {
	return time.Time{}, nil
}
func (f *fakeLockStore) ReleaseLock(_ context.Context, _, _, _ string) error { return nil }
func (f *fakeLockStore) SweepExpiredLocks(_ context.Context) ([]store.LockKey, error) {
	return nil, nil
}

// --- test helpers ---

// spiffeCtx returns a context with a fake mTLS peer carrying the given SPIFFE ID.
func spiffeCtx(spiffeID string) context.Context {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	u, _ := url.Parse(spiffeID)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		URIs:         []*url.URL{u},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	cert, _ := x509.ParseCertificate(der)
	state := tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}}
	p := &peer.Peer{AuthInfo: credentials.TLSInfo{State: state}}
	return peer.NewContext(context.Background(), p)
}

// buildSecret creates a store.Secret with real envelope encryption for tests.
func buildSecret(t *testing.T, masterKey []byte, plaintext []byte) (*store.Secret, error) {
	t.Helper()
	dek, err := icrypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	ct, err := icrypto.Encrypt(dek, plaintext)
	if err != nil {
		return nil, err
	}
	encDEK, err := icrypto.WrapKey(masterKey, dek)
	if err != nil {
		return nil, err
	}
	zeroBytes(dek)
	return &store.Secret{
		Namespace:    "ns",
		Service:      "svc",
		Name:         "key",
		Version:      1,
		Ciphertext:   ct,
		EncryptedDEK: encDEK,
	}, nil
}

// --- GetSecret tests ---

func TestGetSecret_MissingFields(t *testing.T) {
	srv := NewSecretsServer(&fakeSecretFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}))
	tests := []struct {
		name string
		req  *signetv1.GetSecretRequest
	}{
		{"no namespace", &signetv1.GetSecretRequest{Service: "svc", Name: "k"}},
		{"no service", &signetv1.GetSecretRequest{Namespace: "ns", Name: "k"}},
		{"no name", &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := srv.GetSecret(context.Background(), tc.req)
			if status.Code(err) != codes.InvalidArgument {
				t.Errorf("want InvalidArgument, got %v", err)
			}
		})
	}
}

func TestGetSecret_NoMTLS_Unauthenticated(t *testing.T) {
	rec := &fakeRecorder{}
	srv := NewSecretsServer(&fakeSecretFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}))
	_, err := srv.GetSecret(context.Background(), &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "k"})
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("want Unauthenticated, got %v", err)
	}
	if got := rec.last().Outcome; got != "denied" {
		t.Errorf("audit outcome = %q, want denied", got)
	}
	if got := rec.last().SPIFFEID; got != unknownIdentity {
		t.Errorf("audit SPIFFEID = %q, want %q", got, unknownIdentity)
	}
}

func TestGetSecret_PolicyDenied(t *testing.T) {
	rec := &fakeRecorder{}
	checker := &fakeChecker{err: auth.ErrUnauthorized}
	srv := NewSecretsServer(&fakeSecretFetcher{}, &fakeKeyUnwrapper{}, checker, rec, NewBus(), NewLockManager(&fakeLockStore{}))
	ctx := spiffeCtx("spiffe://example.org/workload")
	_, err := srv.GetSecret(ctx, &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "k"})
	if status.Code(err) != codes.PermissionDenied {
		t.Errorf("want PermissionDenied, got %v", err)
	}
	if got := rec.last().Outcome; got != "denied" {
		t.Errorf("audit outcome = %q, want denied", got)
	}
}

func TestGetSecret_SecretNotFound(t *testing.T) {
	rec := &fakeRecorder{}
	fetcher := &fakeSecretFetcher{err: store.ErrNotFound}
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}))
	ctx := spiffeCtx("spiffe://example.org/workload")
	_, err := srv.GetSecret(ctx, &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "k"})
	if status.Code(err) != codes.NotFound {
		t.Errorf("want NotFound, got %v", err)
	}
	if got := rec.last().Outcome; got != "denied" {
		t.Errorf("audit outcome = %q, want denied", got)
	}
}

func TestGetSecret_Sealed(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	sec, err := buildSecret(t, masterKey, []byte("val"))
	if err != nil {
		t.Fatal(err)
	}
	fetcher := &fakeSecretFetcher{secret: sec}
	// Simulate sealed: Use returns ErrKeyNotSet without calling fn.
	unwrapper := &fakeKeyUnwrapper{err: icrypto.ErrKeyNotSet}
	srv := NewSecretsServer(fetcher, unwrapper, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}))
	ctx := spiffeCtx("spiffe://example.org/workload")
	_, err = srv.GetSecret(ctx, &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "key"})
	if status.Code(err) != codes.Unavailable {
		t.Errorf("want Unavailable (sealed), got %v", err)
	}
}

func TestGetSecret_Success(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	want := []byte("my secret value")
	sec, err := buildSecret(t, masterKey, want)
	if err != nil {
		t.Fatal(err)
	}
	rec := &fakeRecorder{}
	fetcher := &fakeSecretFetcher{secret: sec}
	unwrapper := &fakeKeyUnwrapper{key: masterKey}
	srv := NewSecretsServer(fetcher, unwrapper, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}))
	ctx := spiffeCtx("spiffe://example.org/workload")
	resp, err := srv.GetSecret(ctx, &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.Value) != string(want) {
		t.Errorf("value = %q, want %q", resp.Value, want)
	}
	if resp.Version != 1 {
		t.Errorf("version = %d, want 1", resp.Version)
	}
	if got := rec.last().Outcome; got != "permitted" {
		t.Errorf("audit outcome = %q, want permitted", got)
	}
}

func TestGetSecret_AuditIncludesSpiffeID(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	sec, _ := buildSecret(t, masterKey, []byte("val"))
	rec := &fakeRecorder{}
	srv := NewSecretsServer(&fakeSecretFetcher{secret: sec}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}))
	const id = "spiffe://example.org/myservice"
	srv.GetSecret(spiffeCtx(id), &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "key"}) //nolint:errcheck
	if got := rec.last().SPIFFEID; got != id {
		t.Errorf("SPIFFEID = %q, want %q", got, id)
	}
}

func TestGetSecret_CorruptedDataReturnsInternal(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	sec, _ := buildSecret(t, masterKey, []byte("val"))
	// Corrupt the ciphertext — decryption will return ErrAuthenticationFailed.
	sec.Ciphertext[len(sec.Ciphertext)-1] ^= 0xFF
	srv := NewSecretsServer(&fakeSecretFetcher{secret: sec}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}))
	_, err := srv.GetSecret(spiffeCtx("spiffe://x/y"), &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "key"})
	if status.Code(err) != codes.Internal {
		t.Errorf("want Internal (corrupt data), got %v", err)
	}
}

// --- WatchSecret tests ---

func TestWatchSecret_MissingFields(t *testing.T) {
	srv := NewSecretsServer(&fakeSecretFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := newFakeStream(ctx)
	err := srv.WatchSecret(&signetv1.WatchSecretRequest{Service: "svc", Name: "k"}, stream)
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestWatchSecret_InitialSend(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	want := []byte("watch-value")
	sec, _ := buildSecret(t, masterKey, want)
	srv := NewSecretsServer(&fakeSecretFetcher{secret: sec}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}))

	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	stream := newFakeStream(ctx)

	done := make(chan error, 1)
	go func() { done <- srv.WatchSecret(&signetv1.WatchSecretRequest{Namespace: "ns", Service: "svc", Name: "key"}, stream) }()

	select {
	case msg := <-stream.sends:
		if string(msg.Value) != string(want) {
			t.Errorf("value = %q, want %q", msg.Value, want)
		}
		if msg.EventType != signetv1.WatchSecretResponse_EVENT_TYPE_UPDATED {
			t.Errorf("event type = %v, want UPDATED", msg.EventType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial send")
	}

	cancel()
	<-done
}

func TestWatchSecret_ContextCancelTerminates(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	sec, _ := buildSecret(t, masterKey, []byte("v"))
	srv := NewSecretsServer(&fakeSecretFetcher{secret: sec}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}))

	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	stream := newFakeStream(ctx)

	done := make(chan error, 1)
	go func() { done <- srv.WatchSecret(&signetv1.WatchSecretRequest{Namespace: "ns", Service: "svc", Name: "key"}, stream) }()

	// Drain initial send.
	select {
	case <-stream.sends:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial send")
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("want context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for goroutine to exit")
	}
}

func TestWatchSecret_BusNotificationTriggersSend(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	sec1, _ := buildSecret(t, masterKey, []byte("v1"))
	fetcher := &fakeSecretFetcher{secret: sec1}

	bus := NewBus()
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, bus, NewLockManager(&fakeLockStore{}))

	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	defer cancel()
	stream := newFakeStream(ctx)

	done := make(chan error, 1)
	go func() { done <- srv.WatchSecret(&signetv1.WatchSecretRequest{Namespace: "ns", Service: "svc", Name: "key"}, stream) }()

	// Drain initial send.
	select {
	case <-stream.sends:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial send")
	}

	// Update to v2 and notify.
	sec2, _ := buildSecret(t, masterKey, []byte("v2"))
	sec2.Version = 2
	fetcher.mu.Lock()
	fetcher.secret = sec2
	fetcher.mu.Unlock()
	bus.Notify("ns", "svc", "key")

	select {
	case msg := <-stream.sends:
		if msg.Version != 2 {
			t.Errorf("version = %d, want 2", msg.Version)
		}
		if string(msg.Value) != "v2" {
			t.Errorf("value = %q, want v2", msg.Value)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for update send")
	}

	cancel()
	<-done
}

func TestWatchSecret_SecretDeletedSendsDeleteEvent(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	sec, _ := buildSecret(t, masterKey, []byte("v"))
	fetcher := &fakeSecretFetcher{secret: sec}

	bus := NewBus()
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, bus, NewLockManager(&fakeLockStore{}))

	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	defer cancel()
	stream := newFakeStream(ctx)

	done := make(chan error, 1)
	go func() { done <- srv.WatchSecret(&signetv1.WatchSecretRequest{Namespace: "ns", Service: "svc", Name: "key"}, stream) }()

	// Drain initial send.
	select {
	case <-stream.sends:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial send")
	}

	// Simulate deletion.
	fetcher.mu.Lock()
	fetcher.secret = nil
	fetcher.err = store.ErrNotFound
	fetcher.mu.Unlock()
	bus.Notify("ns", "svc", "key")

	select {
	case msg := <-stream.sends:
		if msg.EventType != signetv1.WatchSecretResponse_EVENT_TYPE_DELETED {
			t.Errorf("event type = %v, want DELETED", msg.EventType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for DELETE event")
	}

	select {
	case err := <-done:
		if status.Code(err) != codes.NotFound {
			t.Errorf("stream error = %v, want NotFound", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stream to close")
	}
}

func TestWatchSecret_AuthFailureDenied(t *testing.T) {
	srv := NewSecretsServer(&fakeSecretFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}))
	// No peer in context → Unauthenticated.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := newFakeStream(ctx)
	err := srv.WatchSecret(&signetv1.WatchSecretRequest{Namespace: "ns", Service: "svc", Name: "k"}, stream)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("want Unauthenticated, got %v", err)
	}
}

// --- GetServiceBundle tests ---

// fakeBundleStream implements grpc.ServerStreamingServer[signetv1.WatchServiceBundleResponse].
type fakeBundleStream struct {
	ctx   context.Context
	sends chan *signetv1.WatchServiceBundleResponse
}

func newFakeBundleStream(ctx context.Context) *fakeBundleStream {
	return &fakeBundleStream{ctx: ctx, sends: make(chan *signetv1.WatchServiceBundleResponse, 16)}
}

func (f *fakeBundleStream) Send(r *signetv1.WatchServiceBundleResponse) error {
	f.sends <- r
	return nil
}
func (f *fakeBundleStream) Context() context.Context      { return f.ctx }
func (f *fakeBundleStream) SetHeader(metadata.MD) error   { return nil }
func (f *fakeBundleStream) SendHeader(metadata.MD) error  { return nil }
func (f *fakeBundleStream) SetTrailer(metadata.MD)        {}
func (f *fakeBundleStream) SendMsg(any) error             { return nil }
func (f *fakeBundleStream) RecvMsg(any) error             { return nil }

// bundleFetcher extends fakeSecretFetcher with controllable config and secret returns.
type bundleFetcher struct {
	config  json.RawMessage
	version int
	secrets []store.Secret
	err     error
}

func (b *bundleFetcher) GetSecret(_ context.Context, _, _, _ string) (*store.Secret, error) {
	return nil, store.ErrNotFound
}
func (b *bundleFetcher) GetServiceConfig(_ context.Context, _, _ string) (json.RawMessage, int, error) {
	if b.config == nil {
		return nil, 0, store.ErrNotFound
	}
	return b.config, b.version, b.err
}
func (b *bundleFetcher) FetchServiceSecrets(_ context.Context, _, _ string) ([]store.Secret, error) {
	return b.secrets, b.err
}

func TestGetServiceBundle_MissingFields(t *testing.T) {
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}))
	ctx := spiffeCtx("spiffe://example.org/workload")
	_, err := srv.GetServiceBundle(ctx, &signetv1.GetServiceBundleRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestGetServiceBundle_NoConfigNoSecrets(t *testing.T) {
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}))
	ctx := spiffeCtx("spiffe://example.org/workload")
	resp, err := srv.GetServiceBundle(ctx, &signetv1.GetServiceBundleRequest{Namespace: "ns", Service: "svc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ConfigVersion != 0 {
		t.Errorf("want config_version=0, got %d", resp.ConfigVersion)
	}
	if resp.Bundle == nil {
		t.Fatal("bundle must not be nil")
	}
	fields := resp.Bundle.AsMap()
	secs, ok := fields["secrets"]
	if !ok {
		t.Fatal("bundle must have 'secrets' key")
	}
	if len(secs.(map[string]interface{})) != 0 {
		t.Errorf("expected empty secrets map, got %v", secs)
	}
}

func TestGetServiceBundle_ConfigMergedWithSecrets(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	sec, err := buildSecret(t, masterKey, []byte("topsecret"))
	if err != nil {
		t.Fatal(err)
	}
	fetcher := &bundleFetcher{
		config:  json.RawMessage(`{"port":8080,"db":{"host":"pg"}}`),
		version: 3,
		secrets: []store.Secret{*sec},
	}
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}))
	ctx := spiffeCtx("spiffe://example.org/workload")
	resp, err := srv.GetServiceBundle(ctx, &signetv1.GetServiceBundleRequest{Namespace: "ns", Service: "svc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ConfigVersion != 3 {
		t.Errorf("want config_version=3, got %d", resp.ConfigVersion)
	}
	m := resp.Bundle.AsMap()
	if m["port"] != float64(8080) {
		t.Errorf("want port=8080, got %v", m["port"])
	}
	secsMap, ok := m["secrets"].(map[string]interface{})
	if !ok {
		t.Fatalf("secrets must be a map, got %T", m["secrets"])
	}
	if len(secsMap) != 1 {
		t.Errorf("want 1 secret, got %d", len(secsMap))
	}
	// Value must be non-empty base64 string.
	encoded, ok := secsMap["key"].(string)
	if !ok || encoded == "" {
		t.Errorf("expected base64 string for secret value, got %v", secsMap["key"])
	}
}

// --- WatchServiceBundle tests ---

func TestWatchServiceBundle_MissingFields(t *testing.T) {
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stream := newFakeBundleStream(ctx)
	err := srv.WatchServiceBundle(&signetv1.WatchServiceBundleRequest{}, stream)
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestWatchServiceBundle_ContextCancelTerminates(t *testing.T) {
	bus := NewBus()
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, bus, NewLockManager(&fakeLockStore{}))
	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://example.org/workload"))
	stream := newFakeBundleStream(ctx)
	done := make(chan error, 1)
	go func() {
		done <- srv.WatchServiceBundle(&signetv1.WatchServiceBundleRequest{Namespace: "ns", Service: "svc"}, stream)
	}()
	cancel()
	if err := <-done; err != context.Canceled {
		t.Errorf("want context.Canceled, got %v", err)
	}
}

func TestWatchServiceBundle_NotificationTriggersSend(t *testing.T) {
	bus := NewBus()
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, bus, NewLockManager(&fakeLockStore{}))
	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://example.org/workload"))
	defer cancel()
	stream := newFakeBundleStream(ctx)
	go srv.WatchServiceBundle(&signetv1.WatchServiceBundleRequest{Namespace: "ns", Service: "svc"}, stream) //nolint:errcheck
	// Allow the watcher goroutine to reach its select before we fire.
	time.Sleep(10 * time.Millisecond)
	bus.NotifyBundle("ns", "svc")
	select {
	case msg := <-stream.sends:
		if msg.EventType != signetv1.WatchServiceBundleResponse_EVENT_TYPE_CHANGED {
			t.Errorf("want EVENT_TYPE_CHANGED, got %v", msg.EventType)
		}
		if msg.Namespace != "ns" || msg.Service != "svc" {
			t.Errorf("unexpected namespace/service: %s/%s", msg.Namespace, msg.Service)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for bundle notification")
	}
}
