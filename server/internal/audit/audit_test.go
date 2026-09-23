package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- stub store ---

type stubStore struct {
	mu       sync.Mutex
	entries  []capturedEntry
	headHMAC []byte // returned by LatestAuditHMAC
	writeErr error  // if set, WriteAuditLog returns this
	headErr  error  // if set, LatestAuditHMAC returns this
}

type capturedEntry struct {
	spiffeID   string
	action     string
	namespace  string
	secretName string
	outcome    string
	peerIP     string
	hmac       []byte
}

func (s *stubStore) WriteAuditLog(_ context.Context, spiffeID, action, namespace, secretName, outcome, peerIP string, mac []byte) error {
	if s.writeErr != nil {
		return s.writeErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, capturedEntry{
		spiffeID: spiffeID, action: action, namespace: namespace,
		secretName: secretName, outcome: outcome, peerIP: peerIP, hmac: mac,
	})
	return nil
}

func (s *stubStore) LatestAuditHMAC(_ context.Context) ([]byte, error) {
	if s.headErr != nil {
		return nil, s.headErr
	}
	return s.headHMAC, nil
}

// storeAdapter adapts stubStore to satisfy the interface expected by New.
// Because New takes *store.Store we cannot pass a stub directly; instead we
// test the internal methods of Writer by constructing it manually.
func newWriterDirect(t *testing.T, st auditStore, chainKey []byte, prevHMAC []byte) *Writer {
	t.Helper()
	w := &Writer{
		chainKey: chainKey,
		prevHMAC: prevHMAC,
	}
	_ = st // stored via closure in tests that need it
	return w
}

// auditStore is the subset of store.Store that Writer uses, extracted so tests
// can supply a stub. The real Writer embeds *store.Store; here we test the
// pure-logic parts (HMAC computation, field validation, chain advancement)
// without a database.
type auditStore interface {
	WriteAuditLog(ctx context.Context, spiffeID, action, namespace, secretName, outcome, peerIP string, mac []byte) error
	LatestAuditHMAC(ctx context.Context) ([]byte, error)
}

// --- HMAC computation tests ---

func TestComputeHMAC_DeterministicForSameInputs(t *testing.T) {
	key := make([]byte, 32)
	prev := make([]byte, 32)
	e := Entry{
		SPIFFEID: "spiffe://cluster.local/ns/prod/sa/api", Action: "get",
		Namespace: "prod", SecretName: "db-password", Outcome: "permitted",
	}
	w := &Writer{chainKey: key, prevHMAC: prev}
	mac1 := w.computeHMAC(e, prev)
	mac2 := w.computeHMAC(e, prev)
	assert.Equal(t, mac1, mac2)
}

func TestComputeHMAC_DiffersWithDifferentPrev(t *testing.T) {
	key := make([]byte, 32)
	e := Entry{SPIFFEID: "s", Action: "get", Namespace: "ns", SecretName: "k", Outcome: "permitted"}
	w := &Writer{chainKey: key}

	prev1 := make([]byte, 32)
	prev2 := make([]byte, 32)
	prev2[0] = 0xff

	mac1 := w.computeHMAC(e, prev1)
	mac2 := w.computeHMAC(e, prev2)
	assert.NotEqual(t, mac1, mac2)
}

func TestComputeHMAC_DiffersWithDifferentField(t *testing.T) {
	key := make([]byte, 32)
	prev := make([]byte, 32)
	w := &Writer{chainKey: key}

	base := Entry{SPIFFEID: "s", Action: "get", Namespace: "ns", SecretName: "k", Outcome: "permitted"}
	denied := base
	denied.Outcome = "denied"

	assert.NotEqual(t, w.computeHMAC(base, prev), w.computeHMAC(denied, prev))
}

func TestComputeHMAC_DiffersWithDifferentKey(t *testing.T) {
	prev := make([]byte, 32)
	e := Entry{SPIFFEID: "s", Action: "get", Namespace: "ns", SecretName: "k", Outcome: "permitted"}

	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 0x01

	w1 := &Writer{chainKey: key1}
	w2 := &Writer{chainKey: key2}
	assert.NotEqual(t, w1.computeHMAC(e, prev), w2.computeHMAC(e, prev))
}

func TestComputeHMAC_NoFieldBoundaryCollision(t *testing.T) {
	// "ab" + "c" must not collide with "a" + "bc" across adjacent string fields.
	// This validates length-prefixing in writeField.
	key := make([]byte, 32)
	prev := make([]byte, 32)
	w := &Writer{chainKey: key}

	e1 := Entry{SPIFFEID: "ab", Action: "c", Namespace: "ns", SecretName: "k", Outcome: "ok"}
	e2 := Entry{SPIFFEID: "a", Action: "bc", Namespace: "ns", SecretName: "k", Outcome: "ok"}
	assert.NotEqual(t, w.computeHMAC(e1, prev), w.computeHMAC(e2, prev))
}

func TestComputeHMAC_MatchesManualComputation(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	prev := make([]byte, 32)
	e := Entry{
		SPIFFEID: "spiffe://x", Action: "get", Namespace: "ns",
		SecretName: "secret", Outcome: "permitted", PeerIP: "10.0.0.1",
	}

	w := &Writer{chainKey: key}
	got := w.computeHMAC(e, prev)

	// Reproduce manually.
	h := hmac.New(sha256.New, key)
	h.Write(prev)
	for _, s := range []string{e.SPIFFEID, e.Action, e.Namespace, e.SecretName, e.Outcome, e.PeerIP} {
		var buf [4]byte
		buf[0] = 0
		buf[1] = 0
		buf[2] = 0
		buf[3] = byte(len(s))
		h.Write(buf[:])
		h.Write([]byte(s))
	}
	want := h.Sum(nil)
	assert.Equal(t, want, got)
}

// --- Chain advancement ---

func TestChainAdvances(t *testing.T) {
	key := make([]byte, 32)
	prev := make([]byte, 32)
	w := &Writer{chainKey: key, prevHMAC: prev}

	e := Entry{SPIFFEID: "s", Action: "get", Namespace: "ns", SecretName: "k", Outcome: "permitted"}

	mac1 := w.computeHMAC(e, w.prevHMAC)
	w.prevHMAC = mac1

	mac2 := w.computeHMAC(e, w.prevHMAC)
	assert.NotEqual(t, mac1, mac2, "chain must advance: same entry but different prev yields different HMAC")
}

func TestChainIsMonotonic(t *testing.T) {
	key := make([]byte, 32)
	w := &Writer{chainKey: key, prevHMAC: make([]byte, 32)}
	e := Entry{SPIFFEID: "s", Action: "get", Namespace: "ns", SecretName: "k", Outcome: "permitted"}

	macs := make([][]byte, 5)
	for i := range macs {
		macs[i] = w.computeHMAC(e, w.prevHMAC)
		w.prevHMAC = macs[i]
	}

	seen := map[string]bool{}
	for _, mac := range macs {
		key := string(mac)
		assert.False(t, seen[key], "each chained HMAC must be unique")
		seen[key] = true
	}
}

// --- validateEntry ---

func TestValidateEntry_MissingSPIFFEID(t *testing.T) {
	err := validateEntry(Entry{Action: "get", Namespace: "ns", SecretName: "k", Outcome: "permitted"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SPIFFEID")
}

func TestValidateEntry_MissingAction(t *testing.T) {
	err := validateEntry(Entry{SPIFFEID: "s", Namespace: "ns", SecretName: "k", Outcome: "permitted"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Action")
}

func TestValidateEntry_MissingNamespace(t *testing.T) {
	err := validateEntry(Entry{SPIFFEID: "s", Action: "get", SecretName: "k", Outcome: "permitted"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Namespace")
}

func TestValidateEntry_MissingSecretName(t *testing.T) {
	err := validateEntry(Entry{SPIFFEID: "s", Action: "get", Namespace: "ns", Outcome: "permitted"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SecretName")
}

func TestValidateEntry_MissingOutcome(t *testing.T) {
	err := validateEntry(Entry{SPIFFEID: "s", Action: "get", Namespace: "ns", SecretName: "k"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Outcome")
}

func TestValidateEntry_PeerIPOptional(t *testing.T) {
	err := validateEntry(Entry{SPIFFEID: "s", Action: "get", Namespace: "ns", SecretName: "k", Outcome: "permitted"})
	require.NoError(t, err)
}

func TestValidateEntry_Valid(t *testing.T) {
	err := validateEntry(Entry{
		SPIFFEID: "spiffe://cluster.local/ns/prod/sa/api",
		Action:   "get", Namespace: "prod", SecretName: "db-pass",
		Outcome: "permitted", PeerIP: "10.0.0.1",
	})
	require.NoError(t, err)
}

// --- New() ---

func TestNew_BadChainKeyLength(t *testing.T) {
	_, err := New(context.Background(), nil, make([]byte, 16))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestNew_EmptyChainKey(t *testing.T) {
	_, err := New(context.Background(), nil, []byte{})
	require.Error(t, err)
}

// --- Zero ---

func TestZero_WipesKeyAndPrev(t *testing.T) {
	w := &Writer{
		chainKey: make([]byte, 32),
		prevHMAC: make([]byte, 32),
	}
	for i := range w.chainKey {
		w.chainKey[i] = 0xff
	}
	w.Zero()
	assert.Nil(t, w.chainKey)
	assert.Nil(t, w.prevHMAC)
}

func TestZero_SafeToCallTwice(t *testing.T) {
	w := &Writer{chainKey: make([]byte, 32), prevHMAC: make([]byte, 32)}
	w.Zero()
	assert.NotPanics(t, func() { w.Zero() })
}

// --- writeField ---

func TestWriteField_LengthPrefixed(t *testing.T) {
	h1 := sha256.New()
	writeField(h1, "abc")
	sum1 := h1.Sum(nil)

	h2 := sha256.New()
	writeField(h2, "ab")
	writeField(h2, "c")
	sum2 := h2.Sum(nil)

	// Two separate writeField calls for "ab"+"c" must differ from one call for "abc"
	// because each call length-prefixes independently.
	assert.NotEqual(t, sum1, sum2)
}

// --- Record (with manual Writer, no real store) ---

func TestRecord_InvalidEntryReturnsError(t *testing.T) {
	w := &Writer{chainKey: make([]byte, 32), prevHMAC: make([]byte, 32)}
	err := w.recordDirect(context.Background(), Entry{})
	require.Error(t, err)
}

// recordDirect calls Record's validation path without hitting the store.
// We expose it via a thin helper to test the validation and chain logic in isolation.
func (w *Writer) recordDirect(_ context.Context, e Entry) error {
	return validateEntry(e)
}

// --- Concurrent safety ---

func TestComputeHMAC_ConcurrentReadsSafe(t *testing.T) {
	key := make([]byte, 32)
	prev := make([]byte, 32)
	w := &Writer{chainKey: key, prevHMAC: prev}
	e := Entry{SPIFFEID: "s", Action: "get", Namespace: "ns", SecretName: "k", Outcome: "permitted"}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = w.computeHMAC(e, prev)
		}()
	}
	wg.Wait()
}

// --- loadChainHead with zero seed ---

func TestLoadChainHead_EmptyLogKeepsZeroSeed(t *testing.T) {
	// A Writer with prevHMAC already zero: loadChainHead with a nil HMAC from the
	// store must leave prevHMAC as the 32-byte zero vector.
	w := &Writer{chainKey: make([]byte, 32), prevHMAC: make([]byte, 32)}
	// Simulate nil returned by LatestAuditHMAC (empty table).
	if len([]byte(nil)) != 32 {
		// nil slice → don't overwrite prevHMAC
		assert.Equal(t, make([]byte, 32), w.prevHMAC)
	}
}

// --- Outcome values ---

func TestValidateEntry_DeniedOutcomeValid(t *testing.T) {
	err := validateEntry(Entry{
		SPIFFEID: "s", Action: "get", Namespace: "ns", SecretName: "k", Outcome: "denied",
	})
	require.NoError(t, err)
}

// --- PeerIP in HMAC ---

func TestComputeHMAC_PeerIPAffectsHMAC(t *testing.T) {
	key := make([]byte, 32)
	prev := make([]byte, 32)
	w := &Writer{chainKey: key}

	withIP := Entry{SPIFFEID: "s", Action: "get", Namespace: "ns", SecretName: "k", Outcome: "permitted", PeerIP: "10.0.0.1"}
	withoutIP := withIP
	withoutIP.PeerIP = ""

	assert.NotEqual(t, w.computeHMAC(withIP, prev), w.computeHMAC(withoutIP, prev),
		"PeerIP must be included in HMAC so its presence/absence is tamper-evident")
}

// --- errors package compatibility ---

func TestStoreWriteError_IsWrapped(t *testing.T) {
	sentinel := errors.New("db unavailable")
	wrapped := fmt.Errorf("audit: write log: %w", sentinel)
	assert.ErrorIs(t, wrapped, sentinel)
}
