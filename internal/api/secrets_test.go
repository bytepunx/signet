package api

import (
	"bytes"
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

	signetv1 "github.com/bytepunx/signet/gen/signet/v1"
	"github.com/bytepunx/signet/internal/audit"
	"github.com/bytepunx/signet/internal/auth"
	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	kek    *store.KEK // returned by GetKEKByID when its ID matches
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

func (f *fakeSecretFetcher) GetKEKByID(_ context.Context, id string) (*store.KEK, error) {
	if f.kek != nil && f.kek.ID == id {
		return f.kek, nil
	}
	return nil, store.ErrNotFound
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
func (f *fakeWatchStream) Context() context.Context     { return f.ctx }
func (f *fakeWatchStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeWatchStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeWatchStream) SetTrailer(metadata.MD)       {}
func (f *fakeWatchStream) SendMsg(any) error            { return nil }
func (f *fakeWatchStream) RecvMsg(any) error            { return nil }

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
	ct, err := icrypto.Encrypt(dek, plaintext, nil)
	if err != nil {
		return nil, err
	}
	encDEK, err := icrypto.WrapKey(masterKey, dek, nil)
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

// buildKEK creates a store.KEK wrapped under masterKey, matching the current
// production write path (DEKs are wrapped under an active KEK, not the
// master key directly). Returns the record plus the plaintext KEK bytes so
// callers can build secrets under it via buildSecretUnderKEK.
func buildKEK(t *testing.T, masterKey []byte, id string) (*store.KEK, []byte) {
	t.Helper()
	kekBytes, err := icrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := icrypto.WrapKey(masterKey, kekBytes, icrypto.BindAAD(icrypto.AADKEK))
	if err != nil {
		t.Fatal(err)
	}
	return &store.KEK{ID: id, WrappedKEK: wrapped, IsActive: true}, kekBytes
}

// buildSecretUnderKEK creates a store.Secret whose DEK is wrapped under the
// given KEK and bound via AAD to (namespace, service, name).
func buildSecretUnderKEK(t *testing.T, kekID string, kekBytes []byte, namespace, service, name string, plaintext []byte) *store.Secret {
	t.Helper()
	aad := icrypto.BindAAD(icrypto.AADSecret, namespace, service, name)
	dek, err := icrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	ct, err := icrypto.Encrypt(dek, plaintext, aad)
	if err != nil {
		t.Fatal(err)
	}
	encDEK, err := icrypto.WrapKey(kekBytes, dek, aad)
	if err != nil {
		t.Fatal(err)
	}
	zeroBytes(dek)

	return &store.Secret{
		Namespace:    namespace,
		Service:      service,
		Name:         name,
		Version:      1,
		Ciphertext:   ct,
		EncryptedDEK: encDEK,
		KEKID:        kekID,
	}
}

// --- GetSecret tests ---

func TestGetSecret_MissingFields(t *testing.T) {
	srv := NewSecretsServer(&fakeSecretFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
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
	srv := NewSecretsServer(&fakeSecretFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
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
	srv := NewSecretsServer(&fakeSecretFetcher{}, &fakeKeyUnwrapper{}, checker, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
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
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
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
	srv := NewSecretsServer(fetcher, unwrapper, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
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
	srv := NewSecretsServer(fetcher, unwrapper, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
	ctx := spiffeCtx("spiffe://example.org/workload")
	resp, err := srv.GetSecret(ctx, &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(resp.Value, want) {
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
	srv := NewSecretsServer(&fakeSecretFetcher{secret: sec}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
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
	srv := NewSecretsServer(&fakeSecretFetcher{secret: sec}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
	_, err := srv.GetSecret(spiffeCtx("spiffe://x/y"), &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "key"})
	if status.Code(err) != codes.Internal {
		t.Errorf("want Internal (corrupt data), got %v", err)
	}
}

func TestGetSecret_KEKWrappedSuccess(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	kek, kekBytes := buildKEK(t, masterKey, "kek-1")
	want := []byte("kek wrapped secret")
	sec := buildSecretUnderKEK(t, kek.ID, kekBytes, "ns", "svc", "key", want)

	srv := NewSecretsServer(&fakeSecretFetcher{secret: sec, kek: kek}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
	ctx := spiffeCtx("spiffe://example.org/workload")
	resp, err := srv.GetSecret(ctx, &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "key"})
	require.NoError(t, err)
	assert.Equal(t, want, resp.Value)
}

func TestGetSecret_KEKWrapped_UnknownKEKReturnsError(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	kek, kekBytes := buildKEK(t, masterKey, "kek-1")
	sec := buildSecretUnderKEK(t, kek.ID, kekBytes, "ns", "svc", "key", []byte("val"))

	// fakeSecretFetcher has no matching KEK on file (e.g. it was pruned while
	// still referenced — an operator error, not something that should ever
	// silently decrypt).
	srv := NewSecretsServer(&fakeSecretFetcher{secret: sec}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
	_, err := srv.GetSecret(spiffeCtx("spiffe://x/y"), &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "key"})
	require.Error(t, err)
}

// TestGetSecret_CrossRowBlobSwapFailsAuthentication is the H-1 regression
// test: a party with database write access (but no key material) copies one
// secret's (encrypted_dek, ciphertext) into another secret's row. Before AAD
// binding, this would decrypt successfully under the destination's DEK/master
// key and silently serve the wrong plaintext to whatever caller is authorized
// for the destination row. With AAD bound to (namespace, service, name), the
// swapped blob must fail authentication instead.
func TestGetSecret_CrossRowBlobSwapFailsAuthentication(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	kek, kekBytes := buildKEK(t, masterKey, "kek-1")

	secretA := buildSecretUnderKEK(t, kek.ID, kekBytes, "payments", "db", "password", []byte("hunter2"))
	secretB := buildSecretUnderKEK(t, kek.ID, kekBytes, "low-priv", "app", "password", []byte("low-priv-value"))

	// Attacker with DB write access swaps A's blob into B's row.
	swapped := &store.Secret{
		Namespace:    secretB.Namespace,
		Service:      secretB.Service,
		Name:         secretB.Name,
		Version:      secretB.Version,
		Ciphertext:   secretA.Ciphertext,
		EncryptedDEK: secretA.EncryptedDEK,
		KEKID:        secretA.KEKID,
	}

	srv := NewSecretsServer(&fakeSecretFetcher{secret: swapped, kek: kek}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
	_, err := srv.GetSecret(spiffeCtx("spiffe://x/y"), &signetv1.GetSecretRequest{Namespace: "low-priv", Service: "app", Name: "password"})
	require.Error(t, err, "swapped blob must not decrypt under the destination row's identity")
	assert.Equal(t, codes.Internal, status.Code(err))
}

// TestGetSecret_LegacyDirectMasterWrap_CrossRowSwapStillFails verifies the
// same protection holds for secrets predating the KEK tier (KEKID empty,
// DEK wrapped directly under the master key) as long as they were written
// after AAD binding was introduced.
func TestGetSecret_LegacyDirectMasterWrap_CrossRowSwapStillFails(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()

	buildLegacyAADSecret := func(namespace, service, name string, plaintext []byte) *store.Secret {
		aad := icrypto.BindAAD(icrypto.AADSecret, namespace, service, name)
		dek, err := icrypto.GenerateKey()
		require.NoError(t, err)
		ct, err := icrypto.Encrypt(dek, plaintext, aad)
		require.NoError(t, err)
		encDEK, err := icrypto.WrapKey(masterKey, dek, aad)
		require.NoError(t, err)
		zeroBytes(dek)
		return &store.Secret{Namespace: namespace, Service: service, Name: name, Version: 1, Ciphertext: ct, EncryptedDEK: encDEK}
	}

	secretA := buildLegacyAADSecret("payments", "db", "password", []byte("hunter2"))
	secretB := buildLegacyAADSecret("low-priv", "app", "password", []byte("low-priv-value"))

	swapped := &store.Secret{
		Namespace: secretB.Namespace, Service: secretB.Service, Name: secretB.Name, Version: secretB.Version,
		Ciphertext: secretA.Ciphertext, EncryptedDEK: secretA.EncryptedDEK,
	}

	srv := NewSecretsServer(&fakeSecretFetcher{secret: swapped}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
	_, err := srv.GetSecret(spiffeCtx("spiffe://x/y"), &signetv1.GetSecretRequest{Namespace: "low-priv", Service: "app", Name: "password"})
	require.Error(t, err)
}

// --- WatchSecret tests ---

func TestWatchSecret_MissingFields(t *testing.T) {
	srv := NewSecretsServer(&fakeSecretFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
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
	srv := NewSecretsServer(&fakeSecretFetcher{secret: sec}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)

	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	stream := newFakeStream(ctx)

	done := make(chan error, 1)
	go func() {
		done <- srv.WatchSecret(&signetv1.WatchSecretRequest{Namespace: "ns", Service: "svc", Name: "key"}, stream)
	}()

	select {
	case msg := <-stream.sends:
		if !bytes.Equal(msg.Value, want) {
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
	srv := NewSecretsServer(&fakeSecretFetcher{secret: sec}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)

	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	stream := newFakeStream(ctx)

	done := make(chan error, 1)
	go func() {
		done <- srv.WatchSecret(&signetv1.WatchSecretRequest{Namespace: "ns", Service: "svc", Name: "key"}, stream)
	}()

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
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, bus, NewLockManager(&fakeLockStore{}), false)

	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	defer cancel()
	stream := newFakeStream(ctx)

	done := make(chan error, 1)
	go func() {
		done <- srv.WatchSecret(&signetv1.WatchSecretRequest{Namespace: "ns", Service: "svc", Name: "key"}, stream)
	}()

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
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, bus, NewLockManager(&fakeLockStore{}), false)

	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://x/y"))
	defer cancel()
	stream := newFakeStream(ctx)

	done := make(chan error, 1)
	go func() {
		done <- srv.WatchSecret(&signetv1.WatchSecretRequest{Namespace: "ns", Service: "svc", Name: "key"}, stream)
	}()

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
	srv := NewSecretsServer(&fakeSecretFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
	// No peer in context → Unauthenticated.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := newFakeStream(ctx)
	err := srv.WatchSecret(&signetv1.WatchSecretRequest{Namespace: "ns", Service: "svc", Name: "k"}, stream)
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("want Unauthenticated, got %v", err)
	}
}

// fakeServiceConfigStream implements grpc.ServerStreamingServer[signetv1.WatchServiceConfigResponse].
type fakeServiceConfigStream struct {
	ctx   context.Context
	sends chan *signetv1.WatchServiceConfigResponse
}

func newFakeServiceConfigStream(ctx context.Context) *fakeServiceConfigStream {
	return &fakeServiceConfigStream{ctx: ctx, sends: make(chan *signetv1.WatchServiceConfigResponse, 16)}
}

func (f *fakeServiceConfigStream) Send(r *signetv1.WatchServiceConfigResponse) error {
	f.sends <- r
	return nil
}
func (f *fakeServiceConfigStream) Context() context.Context     { return f.ctx }
func (f *fakeServiceConfigStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeServiceConfigStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeServiceConfigStream) SetTrailer(metadata.MD)       {}
func (f *fakeServiceConfigStream) SendMsg(any) error            { return nil }
func (f *fakeServiceConfigStream) RecvMsg(any) error            { return nil }

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
func (f *fakeBundleStream) Context() context.Context     { return f.ctx }
func (f *fakeBundleStream) SetHeader(metadata.MD) error  { return nil }
func (f *fakeBundleStream) SendHeader(metadata.MD) error { return nil }
func (f *fakeBundleStream) SetTrailer(metadata.MD)       {}
func (f *fakeBundleStream) SendMsg(any) error            { return nil }
func (f *fakeBundleStream) RecvMsg(any) error            { return nil }

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
func (b *bundleFetcher) GetKEKByID(_ context.Context, _ string) (*store.KEK, error) {
	return nil, store.ErrNotFound
}

func TestGetServiceBundle_MissingFields(t *testing.T) {
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
	ctx := spiffeCtx("spiffe://example.org/workload")
	_, err := srv.GetServiceBundle(ctx, &signetv1.GetServiceBundleRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestGetServiceBundle_NoConfigNoSecrets(t *testing.T) {
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
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
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
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
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, NewBus(), NewLockManager(&fakeLockStore{}), false)
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
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, bus, NewLockManager(&fakeLockStore{}), false)
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
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, &fakeRecorder{}, bus, NewLockManager(&fakeLockStore{}), false)
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

// --- H-2: audit coverage for config/bundle paths ---

func TestGetConfig_RecordsAuditOnPermitted(t *testing.T) {
	fetcher := &bundleFetcher{config: json.RawMessage(`{"port":8080}`), version: 1}
	rec := &fakeRecorder{}
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
	ctx := spiffeCtx("spiffe://example.org/workload")

	_, err := srv.GetConfig(ctx, &signetv1.GetConfigRequest{Namespace: "ns", Service: "svc", Key: "port"})
	require.NoError(t, err)

	last := rec.last()
	assert.Equal(t, "get_config", last.Action)
	assert.Equal(t, "permitted", last.Outcome)
	assert.Equal(t, "port", last.SecretName)
}

func TestGetConfig_RecordsAuditOnDenied(t *testing.T) {
	rec := &fakeRecorder{}
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{err: auth.ErrUnauthorized}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
	ctx := spiffeCtx("spiffe://example.org/workload")

	_, err := srv.GetConfig(ctx, &signetv1.GetConfigRequest{Namespace: "ns", Service: "svc", Key: "port"})
	require.Error(t, err)

	last := rec.last()
	assert.Equal(t, "get_config", last.Action)
	assert.Equal(t, "denied", last.Outcome)
}

func TestGetConfig_FailClosed_AuditWriteFails(t *testing.T) {
	fetcher := &bundleFetcher{config: json.RawMessage(`{"port":8080}`), version: 1}
	rec := &fakeRecorder{err: errors.New("db down")}
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), true)
	ctx := spiffeCtx("spiffe://example.org/workload")

	_, err := srv.GetConfig(ctx, &signetv1.GetConfigRequest{Namespace: "ns", Service: "svc", Key: "port"})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestGetConfig_FailOpen_WhenNotConfiguredFailClosed(t *testing.T) {
	fetcher := &bundleFetcher{config: json.RawMessage(`{"port":8080}`), version: 1}
	rec := &fakeRecorder{err: errors.New("db down")}
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
	ctx := spiffeCtx("spiffe://example.org/workload")

	_, err := srv.GetConfig(ctx, &signetv1.GetConfigRequest{Namespace: "ns", Service: "svc", Key: "port"})
	require.NoError(t, err, "with fail-closed disabled, an audit write failure must not block access")
}

func TestGetServiceConfig_RecordsAuditOnPermitted(t *testing.T) {
	fetcher := &bundleFetcher{config: json.RawMessage(`{"port":8080}`), version: 1}
	rec := &fakeRecorder{}
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
	ctx := spiffeCtx("spiffe://example.org/workload")

	_, err := srv.GetServiceConfig(ctx, &signetv1.GetServiceConfigRequest{Namespace: "ns", Service: "svc"})
	require.NoError(t, err)

	last := rec.last()
	assert.Equal(t, "get_service_config", last.Action)
	assert.Equal(t, "permitted", last.Outcome)
	assert.Equal(t, configAuditName, last.SecretName)
}

func TestGetServiceConfig_RecordsAuditOnDenied(t *testing.T) {
	rec := &fakeRecorder{}
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{err: auth.ErrUnauthorized}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
	ctx := spiffeCtx("spiffe://example.org/workload")

	_, err := srv.GetServiceConfig(ctx, &signetv1.GetServiceConfigRequest{Namespace: "ns", Service: "svc"})
	require.Error(t, err)

	last := rec.last()
	assert.Equal(t, "get_service_config", last.Action)
	assert.Equal(t, "denied", last.Outcome)
}

func TestGetServiceConfig_FailClosed_AuditWriteFails(t *testing.T) {
	fetcher := &bundleFetcher{config: json.RawMessage(`{"port":8080}`), version: 1}
	rec := &fakeRecorder{err: errors.New("db down")}
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), true)
	ctx := spiffeCtx("spiffe://example.org/workload")

	_, err := srv.GetServiceConfig(ctx, &signetv1.GetServiceConfigRequest{Namespace: "ns", Service: "svc"})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestWatchServiceConfig_RecordsAuditOnPermitted(t *testing.T) {
	fetcher := &bundleFetcher{config: json.RawMessage(`{"port":8080}`), version: 1}
	rec := &fakeRecorder{}
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://example.org/workload"))
	defer cancel()
	stream := newFakeServiceConfigStream(ctx)

	done := make(chan error, 1)
	go func() {
		done <- srv.WatchServiceConfig(&signetv1.WatchServiceConfigRequest{Namespace: "ns", Service: "svc"}, stream)
	}()
	select {
	case <-stream.sends:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for initial send")
	}
	cancel()
	<-done

	last := rec.last()
	assert.Equal(t, "watch_service_config", last.Action)
	assert.Equal(t, "permitted", last.Outcome)
}

func TestWatchServiceConfig_FailClosed_AuditWriteFails(t *testing.T) {
	fetcher := &bundleFetcher{config: json.RawMessage(`{"port":8080}`), version: 1}
	rec := &fakeRecorder{err: errors.New("db down")}
	srv := NewSecretsServer(fetcher, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), true)
	ctx := spiffeCtx("spiffe://example.org/workload")
	stream := newFakeServiceConfigStream(ctx)

	err := srv.WatchServiceConfig(&signetv1.WatchServiceConfigRequest{Namespace: "ns", Service: "svc"}, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestGetServiceBundle_RecordsAuditOnPermitted(t *testing.T) {
	rec := &fakeRecorder{}
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
	ctx := spiffeCtx("spiffe://example.org/workload")

	_, err := srv.GetServiceBundle(ctx, &signetv1.GetServiceBundleRequest{Namespace: "ns", Service: "svc"})
	require.NoError(t, err)

	last := rec.last()
	assert.Equal(t, "get_bundle", last.Action)
	assert.Equal(t, "permitted", last.Outcome)
	assert.Equal(t, bundleAuditName, last.SecretName)
}

func TestGetServiceBundle_RecordsAuditOnDenied(t *testing.T) {
	rec := &fakeRecorder{}
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{err: auth.ErrUnauthorized}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
	ctx := spiffeCtx("spiffe://example.org/workload")

	_, err := srv.GetServiceBundle(ctx, &signetv1.GetServiceBundleRequest{Namespace: "ns", Service: "svc"})
	require.Error(t, err)

	last := rec.last()
	assert.Equal(t, "get_bundle", last.Action)
	assert.Equal(t, "denied", last.Outcome)
}

func TestGetServiceBundle_FailClosed_AuditWriteFails(t *testing.T) {
	rec := &fakeRecorder{err: errors.New("db down")}
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), true)
	ctx := spiffeCtx("spiffe://example.org/workload")

	_, err := srv.GetServiceBundle(ctx, &signetv1.GetServiceBundleRequest{Namespace: "ns", Service: "svc"})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestWatchServiceBundle_RecordsAuditOnSubscribe(t *testing.T) {
	rec := &fakeRecorder{}
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)
	ctx, cancel := context.WithCancel(spiffeCtx("spiffe://example.org/workload"))
	stream := newFakeBundleStream(ctx)

	done := make(chan error, 1)
	go func() {
		done <- srv.WatchServiceBundle(&signetv1.WatchServiceBundleRequest{Namespace: "ns", Service: "svc"}, stream)
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	<-done

	last := rec.last()
	assert.Equal(t, "watch_bundle", last.Action)
	assert.Equal(t, "permitted", last.Outcome)
}

func TestWatchServiceBundle_FailClosed_AuditWriteFails(t *testing.T) {
	rec := &fakeRecorder{err: errors.New("db down")}
	srv := NewSecretsServer(&bundleFetcher{}, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), true)
	ctx := spiffeCtx("spiffe://example.org/workload")
	stream := newFakeBundleStream(ctx)

	err := srv.WatchServiceBundle(&signetv1.WatchServiceBundleRequest{Namespace: "ns", Service: "svc"}, stream)
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

// --- H-3: fail-closed audit for GetSecret/WatchSecret ---

func TestGetSecret_FailClosed_AuditWriteFails(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	sec, _ := buildSecret(t, masterKey, []byte("val"))
	rec := &fakeRecorder{err: errors.New("db down")}
	srv := NewSecretsServer(&fakeSecretFetcher{secret: sec}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), true)

	_, err := srv.GetSecret(spiffeCtx("spiffe://x/y"), &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "key"})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestGetSecret_FailOpen_WhenNotConfiguredFailClosed(t *testing.T) {
	masterKey, _ := icrypto.GenerateKey()
	sec, _ := buildSecret(t, masterKey, []byte("val"))
	rec := &fakeRecorder{err: errors.New("db down")}
	srv := NewSecretsServer(&fakeSecretFetcher{secret: sec}, &fakeKeyUnwrapper{key: masterKey}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), false)

	resp, err := srv.GetSecret(spiffeCtx("spiffe://x/y"), &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "key"})
	require.NoError(t, err, "with fail-closed disabled, an audit write failure must not block access")
	assert.Equal(t, []byte("val"), resp.Value)
}

func TestGetSecret_DeniedOutcome_NotSubjectToFailClosed(t *testing.T) {
	// A denied access should still return the original denial error, not the
	// fail-closed "audit write failed" error — fail-closed only guards
	// access that was otherwise going to be permitted.
	rec := &fakeRecorder{err: errors.New("db down")}
	srv := NewSecretsServer(&fakeSecretFetcher{err: store.ErrNotFound}, &fakeKeyUnwrapper{}, &fakeChecker{}, rec, NewBus(), NewLockManager(&fakeLockStore{}), true)

	_, err := srv.GetSecret(spiffeCtx("spiffe://x/y"), &signetv1.GetSecretRequest{Namespace: "ns", Service: "svc", Name: "key"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}
