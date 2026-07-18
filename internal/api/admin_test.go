package api

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/bytepunx/signet/internal/auth"
	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
	"github.com/bytepunx/signet/internal/unseal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// --- fakes ---

type fakeUnsealMgr struct {
	unsealKeyErr         error
	submitShareResult    unseal.Status
	submitShareErr       error
	statusResult         unseal.Status
	sealCalled           bool
	rotateMasterKeyErr   error
	rotateMasterKeyCalls [][]byte
}

func (f *fakeUnsealMgr) UnsealWithKey(_ []byte) error { return f.unsealKeyErr }
func (f *fakeUnsealMgr) SubmitShare(_ []byte) (unseal.Status, error) {
	return f.submitShareResult, f.submitShareErr
}
func (f *fakeUnsealMgr) Seal()                 { f.sealCalled = true }
func (f *fakeUnsealMgr) Status() unseal.Status { return f.statusResult }
func (f *fakeUnsealMgr) RotateMasterKey(newKey []byte) error {
	f.rotateMasterKeyCalls = append(f.rotateMasterKeyCalls, newKey)
	return f.rotateMasterKeyErr
}

type fakeTokenChecker struct {
	err error
}

func (f *fakeTokenChecker) Validate(_ context.Context, _ string) error { return f.err }

// fakeAdminStore implements adminStore for testing, actually persisting KEKs
// and the key-check value in memory so rotation flows can be exercised.
type fakeAdminStore struct {
	kcv          []byte
	kcvErr       error
	keks         []store.KEK
	putKEKErr    error
	secretRefs   []store.SecretKeyRef
	rewrapErr    error
	policies     []store.Policy
	putPolicyErr error
}

func (f *fakeAdminStore) GetKeyCheckValue(_ context.Context) ([]byte, error) {
	if f.kcvErr != nil {
		return nil, f.kcvErr
	}
	if f.kcv == nil {
		return nil, store.ErrNotFound
	}
	return f.kcv, nil
}

func (f *fakeAdminStore) PutKeyCheckValue(_ context.Context, ciphertext []byte) error {
	f.kcv = ciphertext
	return nil
}

func (f *fakeAdminStore) GetActiveKEK(_ context.Context) (*store.KEK, error) {
	for _, k := range f.keks {
		if k.IsActive {
			kk := k
			return &kk, nil
		}
	}
	return nil, store.ErrNotFound
}

func (f *fakeAdminStore) GetKEKByID(_ context.Context, id string) (*store.KEK, error) {
	for _, k := range f.keks {
		if k.ID == id {
			kk := k
			return &kk, nil
		}
	}
	return nil, store.ErrNotFound
}

func (f *fakeAdminStore) PutKEK(_ context.Context, k *store.KEK) error {
	if f.putKEKErr != nil {
		return f.putKEKErr
	}
	k.ID = fmt.Sprintf("kek-%d", len(f.keks)+1)
	f.keks = append(f.keks, *k)
	return nil
}

func (f *fakeAdminStore) ListKEKs(_ context.Context) ([]store.KEK, error) {
	return f.keks, nil
}

func (f *fakeAdminStore) DeactivateKEK(_ context.Context, id string) error {
	for i := range f.keks {
		if f.keks[i].ID == id {
			f.keks[i].IsActive = false
			return nil
		}
	}
	return store.ErrNotFound
}

func (f *fakeAdminStore) DeleteKEK(_ context.Context, id string) error {
	for i, k := range f.keks {
		if k.ID == id {
			f.keks = append(f.keks[:i], f.keks[i+1:]...)
			return nil
		}
	}
	return store.ErrNotFound
}

func (f *fakeAdminStore) PutPolicy(_ context.Context, p *store.Policy) error {
	if f.putPolicyErr != nil {
		return f.putPolicyErr
	}
	p.ID = fmt.Sprintf("policy-%d", len(f.policies)+1)
	f.policies = append(f.policies, *p)
	return nil
}

func (f *fakeAdminStore) ListPolicies(_ context.Context) ([]store.Policy, error) {
	return f.policies, nil
}

func (f *fakeAdminStore) DeletePolicy(_ context.Context, id string) error {
	for i, p := range f.policies {
		if p.ID == id {
			f.policies = append(f.policies[:i], f.policies[i+1:]...)
			return nil
		}
	}
	return store.ErrNotFound
}

func (f *fakeAdminStore) CountSecretsUsingKEK(_ context.Context, id string) (int, error) {
	n := 0
	for _, r := range f.secretRefs {
		if r.KEKID == id {
			n++
		}
	}
	return n, nil
}

func (f *fakeAdminStore) ListSecretKeyRefs(_ context.Context) ([]store.SecretKeyRef, error) {
	return f.secretRefs, nil
}

func (f *fakeAdminStore) UpdateSecretDEK(_ context.Context, namespace, service, name string, version int, newEncDEK []byte, newKEKID string) error {
	for i := range f.secretRefs {
		r := &f.secretRefs[i]
		if r.Namespace == namespace && r.Service == service && r.Name == name && r.Version == version {
			r.EncryptedDEK = newEncDEK
			r.KEKID = newKEKID
			return nil
		}
	}
	return store.ErrNotFound
}

func (f *fakeAdminStore) RewrapKEKsAndKCV(_ context.Context, kekUpdates []store.KEKRewrap, newKCV []byte) error {
	if f.rewrapErr != nil {
		return f.rewrapErr
	}
	for _, u := range kekUpdates {
		found := false
		for i := range f.keks {
			if f.keks[i].ID == u.ID {
				f.keks[i].WrappedKEK = u.WrappedKEK
				found = true
			}
		}
		if !found {
			return store.ErrNotFound
		}
	}
	f.kcv = newKCV
	return nil
}

// adminTestKey is a fixed 32-byte "master key" for tests that need Use to
// actually succeed (e.g. any path that reaches the key-check value or KEK
// crypto operations).
var adminTestKey = make([]byte, 32)

// bearerCtx returns a context with an Authorization: Bearer <token> gRPC metadata header.
func bearerCtx(token string) context.Context {
	md := metadata.Pairs("authorization", fmt.Sprintf("Bearer %s", token))
	return metadata.NewIncomingContext(context.Background(), md)
}

// --- requireToken tests ---

func TestAdminServer_NoToken_Unauthenticated(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.Status(context.Background(), &adminv1.StatusRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("want Unauthenticated, got %v", err)
	}
}

func TestAdminServer_InvalidToken_Unauthenticated(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{err: auth.ErrInvalidToken}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.Status(bearerCtx("bad-token"), &adminv1.StatusRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("want Unauthenticated, got %v", err)
	}
}

// --- UnsealKey tests ---

func TestUnsealKey_EmptyKey(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.UnsealKey(bearerCtx("tok"), &adminv1.UnsealKeyRequest{Key: nil})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestUnsealKey_AlreadyUnsealed(t *testing.T) {
	mgr := &fakeUnsealMgr{unsealKeyErr: unseal.ErrAlreadyUnsealed}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.UnsealKey(bearerCtx("tok"), &adminv1.UnsealKeyRequest{Key: []byte("k")})
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("want FailedPrecondition, got %v", err)
	}
}

func TestUnsealKey_Success(t *testing.T) {
	mgr := &fakeUnsealMgr{
		statusResult: unseal.Status{State: unseal.StateUnsealed},
	}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	resp, err := srv.UnsealKey(bearerCtx("tok"), &adminv1.UnsealKeyRequest{Key: make([]byte, 32)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Unsealed {
		t.Error("expected Unsealed = true")
	}
}

// --- UnsealShare tests ---

func TestUnsealShare_EmptyShare(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.UnsealShare(bearerCtx("tok"), &adminv1.UnsealShareRequest{Share: nil})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestUnsealShare_InvalidShare(t *testing.T) {
	mgr := &fakeUnsealMgr{submitShareErr: unseal.ErrInvalidShare}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.UnsealShare(bearerCtx("tok"), &adminv1.UnsealShareRequest{Share: []byte("bad")})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("want InvalidArgument, got %v", err)
	}
}

func TestUnsealShare_SharesExpired(t *testing.T) {
	mgr := &fakeUnsealMgr{submitShareErr: unseal.ErrSharesExpired}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.UnsealShare(bearerCtx("tok"), &adminv1.UnsealShareRequest{Share: []byte("s")})
	if status.Code(err) != codes.DeadlineExceeded {
		t.Errorf("want DeadlineExceeded, got %v", err)
	}
}

func TestUnsealShare_PartialProgress(t *testing.T) {
	mgr := &fakeUnsealMgr{
		submitShareResult: unseal.Status{
			State:          unseal.StateUnsealing,
			SharesReceived: 1,
			SharesRequired: 3,
		},
	}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	resp, err := srv.UnsealShare(bearerCtx("tok"), &adminv1.UnsealShareRequest{Share: []byte("s")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Unsealed {
		t.Error("expected Unsealed = false")
	}
	if resp.SharesReceived != 1 {
		t.Errorf("SharesReceived = %d, want 1", resp.SharesReceived)
	}
	if resp.SharesRequired != 3 {
		t.Errorf("SharesRequired = %d, want 3", resp.SharesRequired)
	}
}

func TestUnsealShare_ThresholdMet(t *testing.T) {
	mgr := &fakeUnsealMgr{
		submitShareResult: unseal.Status{
			State:          unseal.StateUnsealed,
			SharesReceived: 3,
			SharesRequired: 3,
		},
	}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	resp, err := srv.UnsealShare(bearerCtx("tok"), &adminv1.UnsealShareRequest{Share: []byte("s")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Unsealed {
		t.Error("expected Unsealed = true")
	}
}

// --- Seal tests ---

func TestSeal_Success(t *testing.T) {
	mgr := &fakeUnsealMgr{}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	resp, err := srv.Seal(bearerCtx("tok"), &adminv1.SealRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mgr.sealCalled {
		t.Error("expected Seal to be called on manager")
	}
	if resp.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestSeal_NoToken(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.Seal(context.Background(), &adminv1.SealRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Errorf("want Unauthenticated, got %v", err)
	}
}

// --- Status tests ---

func TestStatus_AllStates(t *testing.T) {
	tests := []struct {
		state     unseal.State
		wantProto adminv1.StatusResponse_State
	}{
		{unseal.StateSealed, adminv1.StatusResponse_STATE_SEALED},
		{unseal.StateUnsealing, adminv1.StatusResponse_STATE_UNSEALING},
		{unseal.StateUnsealed, adminv1.StatusResponse_STATE_UNSEALED},
	}
	for _, tc := range tests {
		t.Run(tc.state.String(), func(t *testing.T) {
			mgr := &fakeUnsealMgr{statusResult: unseal.Status{State: tc.state}}
			srv := NewAdminServer(mgr, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
			resp, err := srv.Status(bearerCtx("tok"), &adminv1.StatusRequest{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.State != tc.wantProto {
				t.Errorf("state = %v, want %v", resp.State, tc.wantProto)
			}
		})
	}
}

func TestStatus_ShareProgress(t *testing.T) {
	mgr := &fakeUnsealMgr{
		statusResult: unseal.Status{
			State:          unseal.StateUnsealing,
			SharesReceived: 2,
			SharesRequired: 5,
		},
	}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	resp, err := srv.Status(bearerCtx("tok"), &adminv1.StatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SharesReceived != 2 {
		t.Errorf("SharesReceived = %d, want 2", resp.SharesReceived)
	}
	if resp.SharesRequired != 5 {
		t.Errorf("SharesRequired = %d, want 5", resp.SharesRequired)
	}
}

// --- Key-check value tests ---

func TestUnsealKey_FirstUnseal_MintsKeyCheckValue(t *testing.T) {
	mgr := &fakeUnsealMgr{statusResult: unseal.Status{State: unseal.StateUnsealed}}
	st := &fakeAdminStore{}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})

	_, err := srv.UnsealKey(bearerCtx("tok"), &adminv1.UnsealKeyRequest{Key: make([]byte, 32)})
	require.NoError(t, err)
	assert.NotEmpty(t, st.kcv)
}

func TestUnsealKey_SecondUnsealWithSameKey_Succeeds(t *testing.T) {
	mgr := &fakeUnsealMgr{statusResult: unseal.Status{State: unseal.StateUnsealed}}
	st := &fakeAdminStore{}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})

	_, err := srv.UnsealKey(bearerCtx("tok"), &adminv1.UnsealKeyRequest{Key: make([]byte, 32)})
	require.NoError(t, err)

	// Simulate a restart: same persisted KCV, same key.
	resp, err := srv.UnsealKey(bearerCtx("tok"), &adminv1.UnsealKeyRequest{Key: make([]byte, 32)})
	require.NoError(t, err)
	assert.True(t, resp.Unsealed)
}

func TestUnsealKey_KeyCheckMismatch_ReSealsAndReturnsError(t *testing.T) {
	otherKey := make([]byte, 32)
	otherKey[0] = 0x01 // different from adminTestKey (all zero)
	ct, err := icrypto.Encrypt(otherKey, []byte(kcvPlaintext), icrypto.BindAAD(icrypto.AADKeyCheckValue))
	require.NoError(t, err)

	mgr := &fakeUnsealMgr{statusResult: unseal.Status{State: unseal.StateUnsealed}}
	st := &fakeAdminStore{kcv: ct}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})

	_, err = srv.UnsealKey(bearerCtx("tok"), &adminv1.UnsealKeyRequest{Key: make([]byte, 32)})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.True(t, mgr.sealCalled, "must re-seal when the key does not match the key-check value")
}

func TestUnsealShare_ThresholdMet_KeyCheckMismatch_ReSeals(t *testing.T) {
	otherKey := make([]byte, 32)
	otherKey[0] = 0x01
	ct, err := icrypto.Encrypt(otherKey, []byte(kcvPlaintext), icrypto.BindAAD(icrypto.AADKeyCheckValue))
	require.NoError(t, err)

	mgr := &fakeUnsealMgr{
		submitShareResult: unseal.Status{State: unseal.StateUnsealed, SharesReceived: 3, SharesRequired: 3},
	}
	st := &fakeAdminStore{kcv: ct}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})

	_, err = srv.UnsealShare(bearerCtx("tok"), &adminv1.UnsealShareRequest{Share: []byte("s")})
	require.Error(t, err)
	assert.True(t, mgr.sealCalled)
}

// --- RotateKEK tests ---

func TestRotateKEK_NoToken(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{err: auth.ErrInvalidToken}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.RotateKEK(bearerCtx("bad"), &adminv1.RotateKEKRequest{})
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestRotateKEK_FreshDeployment_CreatesFirstKEK(t *testing.T) {
	st := &fakeAdminStore{}
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})

	resp, err := srv.RotateKEK(bearerCtx("tok"), &adminv1.RotateKEKRequest{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.NewKekId)
	assert.Empty(t, resp.OldKekId)
	assert.Equal(t, int32(0), resp.SecretsRewrapped)
	require.Len(t, st.keks, 1)
	assert.True(t, st.keks[0].IsActive)
}

func TestRotateKEK_RewrapsSecretsReferencingOldKEK(t *testing.T) {
	st := &fakeAdminStore{}
	keys := &fakeKeyUnwrapper{key: adminTestKey}
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, st, keys)

	first, err := srv.RotateKEK(bearerCtx("tok"), &adminv1.RotateKEKRequest{})
	require.NoError(t, err)
	oldKEKID := first.NewKekId

	var oldRec store.KEK
	for _, k := range st.keks {
		if k.ID == oldKEKID {
			oldRec = k
		}
	}
	var oldKEKBytes []byte
	require.NoError(t, keys.Use(func(masterKey []byte) error {
		b, uErr := icrypto.UnwrapKey(masterKey, oldRec.WrappedKEK, icrypto.BindAAD(icrypto.AADKEK))
		oldKEKBytes = b
		return uErr
	}))

	aad := icrypto.BindAAD(icrypto.AADSecret, "ns", "svc", "key")
	dek, err := icrypto.GenerateKey()
	require.NoError(t, err)
	encDEK, err := icrypto.WrapKey(oldKEKBytes, dek, aad)
	require.NoError(t, err)
	st.secretRefs = []store.SecretKeyRef{
		{Namespace: "ns", Service: "svc", Name: "key", Version: 1, EncryptedDEK: encDEK, KEKID: oldKEKID},
	}

	resp, err := srv.RotateKEK(bearerCtx("tok"), &adminv1.RotateKEKRequest{})
	require.NoError(t, err)
	assert.Equal(t, oldKEKID, resp.OldKekId)
	assert.NotEqual(t, oldKEKID, resp.NewKekId)
	assert.Equal(t, int32(1), resp.SecretsRewrapped)

	ref := st.secretRefs[0]
	assert.Equal(t, resp.NewKekId, ref.KEKID)
	assert.NotEqual(t, encDEK, ref.EncryptedDEK)

	var newRec store.KEK
	for _, k := range st.keks {
		if k.ID == resp.NewKekId {
			newRec = k
		}
	}
	var newKEKBytes []byte
	require.NoError(t, keys.Use(func(masterKey []byte) error {
		b, uErr := icrypto.UnwrapKey(masterKey, newRec.WrappedKEK, icrypto.BindAAD(icrypto.AADKEK))
		newKEKBytes = b
		return uErr
	}))
	gotDEK, err := icrypto.UnwrapKey(newKEKBytes, ref.EncryptedDEK, aad)
	require.NoError(t, err)
	assert.Equal(t, dek, gotDEK)

	// The old KEK must be retained (deactivated, not deleted) so it can still
	// be looked up if anything was missed.
	found := false
	for _, k := range st.keks {
		if k.ID == oldKEKID {
			found = true
			assert.False(t, k.IsActive)
		}
	}
	assert.True(t, found, "old kek must be retained, not deleted")
}

// --- ListKEKs tests ---

func TestListKEKs_ReturnsAllWithoutKeyMaterial(t *testing.T) {
	st := &fakeAdminStore{keks: []store.KEK{
		{ID: "k1", WrappedKEK: []byte("wrapped-bytes"), IsActive: true, CreatedAt: time.Now()},
	}}
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})

	resp, err := srv.ListKEKs(bearerCtx("tok"), &adminv1.ListKEKsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Keks, 1)
	assert.Equal(t, "k1", resp.Keks[0].Id)
	assert.True(t, resp.Keks[0].IsActive)
	assert.Empty(t, resp.Keks[0].DeactivatedAt)
}

// --- PruneKEK tests ---

func TestPruneKEK_EmptyID(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.PruneKEK(bearerCtx("tok"), &adminv1.PruneKEKRequest{Id: ""})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestPruneKEK_ActiveKEKRejected(t *testing.T) {
	st := &fakeAdminStore{keks: []store.KEK{{ID: "k1", IsActive: true}}}
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.PruneKEK(bearerCtx("tok"), &adminv1.PruneKEKRequest{Id: "k1"})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestPruneKEK_StillReferencedRejected(t *testing.T) {
	st := &fakeAdminStore{
		keks:       []store.KEK{{ID: "k1", IsActive: false}, {ID: "k2", IsActive: true}},
		secretRefs: []store.SecretKeyRef{{Namespace: "ns", Service: "svc", Name: "n", Version: 1, KEKID: "k1"}},
	}
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.PruneKEK(bearerCtx("tok"), &adminv1.PruneKEKRequest{Id: "k1"})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestPruneKEK_UnreferencedInactiveSucceeds(t *testing.T) {
	st := &fakeAdminStore{keks: []store.KEK{{ID: "k1", IsActive: false}, {ID: "k2", IsActive: true}}}
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.PruneKEK(bearerCtx("tok"), &adminv1.PruneKEKRequest{Id: "k1"})
	require.NoError(t, err)
	assert.Len(t, st.keks, 1)
}

// --- RotateMasterKey tests ---

func TestRotateMasterKey_InvalidKeySize(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.RotateMasterKey(bearerCtx("tok"), &adminv1.RotateMasterKeyRequest{NewKey: []byte("short")})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestRotateMasterKey_NoExistingKeyCheckValueRefused(t *testing.T) {
	// A server that is Unsealed always has a key-check value (minted on first
	// unseal); its absence signals a pre-existing inconsistency, so rotation
	// must refuse rather than compound it.
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.RotateMasterKey(bearerCtx("tok"), &adminv1.RotateMasterKeyRequest{NewKey: make([]byte, 32)})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestRotateMasterKey_RewrapsKEKsAndAdoptsNewKey(t *testing.T) {
	kekBytes, err := icrypto.GenerateKey()
	require.NoError(t, err)
	wrapped, err := icrypto.WrapKey(adminTestKey, kekBytes, icrypto.BindAAD(icrypto.AADKEK))
	require.NoError(t, err)
	oldKCV, err := icrypto.Encrypt(adminTestKey, []byte(kcvPlaintext), icrypto.BindAAD(icrypto.AADKeyCheckValue))
	require.NoError(t, err)

	st := &fakeAdminStore{keks: []store.KEK{{ID: "k1", WrappedKEK: wrapped, IsActive: true}}, kcv: oldKCV}
	mgr := &fakeUnsealMgr{}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})

	newKey := make([]byte, 32)
	newKey[0] = 0x42
	resp, err := srv.RotateMasterKey(bearerCtx("tok"), &adminv1.RotateMasterKeyRequest{NewKey: newKey})
	require.NoError(t, err)
	assert.Equal(t, int32(1), resp.KeksRewrapped)
	require.Len(t, mgr.rotateMasterKeyCalls, 1)
	assert.Equal(t, newKey, mgr.rotateMasterKeyCalls[0])

	gotKEK, err := icrypto.UnwrapKey(newKey, st.keks[0].WrappedKEK, icrypto.BindAAD(icrypto.AADKEK))
	require.NoError(t, err)
	assert.Equal(t, kekBytes, gotKEK)

	kcvPlain, err := icrypto.Decrypt(newKey, st.kcv, icrypto.BindAAD(icrypto.AADKeyCheckValue))
	require.NoError(t, err)
	assert.Equal(t, kcvPlaintext, string(kcvPlain))
}

func TestRotateMasterKey_ManagerFailureRollsBackDB(t *testing.T) {
	kekBytes, err := icrypto.GenerateKey()
	require.NoError(t, err)
	wrapped, err := icrypto.WrapKey(adminTestKey, kekBytes, icrypto.BindAAD(icrypto.AADKEK))
	require.NoError(t, err)
	oldKCV, err := icrypto.Encrypt(adminTestKey, []byte(kcvPlaintext), icrypto.BindAAD(icrypto.AADKeyCheckValue))
	require.NoError(t, err)

	st := &fakeAdminStore{keks: []store.KEK{{ID: "k1", WrappedKEK: wrapped, IsActive: true}}, kcv: oldKCV}
	mgr := &fakeUnsealMgr{rotateMasterKeyErr: errors.New("boom")}
	srv := NewAdminServer(mgr, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})

	newKey := make([]byte, 32)
	newKey[0] = 0x42
	_, err = srv.RotateMasterKey(bearerCtx("tok"), &adminv1.RotateMasterKeyRequest{NewKey: newKey})
	require.Error(t, err)

	// The DB must have been rolled back: the KEK and KCV should still be
	// readable under the OLD (still-loaded) master key, not the new one.
	gotKEK, err := icrypto.UnwrapKey(adminTestKey, st.keks[0].WrappedKEK, icrypto.BindAAD(icrypto.AADKEK))
	require.NoError(t, err)
	assert.Equal(t, kekBytes, gotKEK)

	kcvPlain, err := icrypto.Decrypt(adminTestKey, st.kcv, icrypto.BindAAD(icrypto.AADKeyCheckValue))
	require.NoError(t, err)
	assert.Equal(t, kcvPlaintext, string(kcvPlain))
}

// --- CreatePolicy tests ---

func TestCreatePolicy_RequiresSpiffeID(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.CreatePolicy(bearerCtx("tok"), &adminv1.CreatePolicyRequest{Namespace: "ns", Service: "svc"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreatePolicy_RequiresNamespace(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.CreatePolicy(bearerCtx("tok"), &adminv1.CreatePolicyRequest{SpiffeId: "spiffe://x/y", Service: "svc"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreatePolicy_RequiresService(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.CreatePolicy(bearerCtx("tok"), &adminv1.CreatePolicyRequest{SpiffeId: "spiffe://x/y", Namespace: "ns"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCreatePolicy_DefaultsSecretNameAndPermissions(t *testing.T) {
	st := &fakeAdminStore{}
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})
	resp, err := srv.CreatePolicy(bearerCtx("tok"), &adminv1.CreatePolicyRequest{
		SpiffeId: "spiffe://cluster.local/ns/*/sa/echo", Namespace: "shared", Service: "common",
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Id)
	require.Len(t, st.policies, 1)
	assert.Equal(t, "shared/common/*", st.policies[0].Pattern)
	assert.Equal(t, []string{"get"}, st.policies[0].Permissions)
}

func TestCreatePolicy_ExplicitSecretNameAndPermissions(t *testing.T) {
	st := &fakeAdminStore{}
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.CreatePolicy(bearerCtx("tok"), &adminv1.CreatePolicyRequest{
		SpiffeId: "spiffe://x/y", Namespace: "ns", Service: "svc",
		SecretName: "db-*", Permissions: []string{"get", "list"},
	})
	require.NoError(t, err)
	require.Len(t, st.policies, 1)
	assert.Equal(t, "ns/svc/db-*", st.policies[0].Pattern)
	assert.Equal(t, []string{"get", "list"}, st.policies[0].Permissions)
}

// --- ListPolicies tests ---

func TestListPolicies_ReturnsAll(t *testing.T) {
	st := &fakeAdminStore{policies: []store.Policy{
		{ID: "p1", SPIFFEID: "spiffe://x/y", Namespace: "ns", Pattern: "ns/svc/*", Permissions: []string{"get"}, CreatedAt: time.Now()},
	}}
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})
	resp, err := srv.ListPolicies(bearerCtx("tok"), &adminv1.ListPoliciesRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Policies, 1)
	assert.Equal(t, "p1", resp.Policies[0].Id)
	assert.Equal(t, "ns/svc/*", resp.Policies[0].Pattern)
}

// --- DeletePolicy tests ---

func TestDeletePolicy_EmptyID(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.DeletePolicy(bearerCtx("tok"), &adminv1.DeletePolicyRequest{Id: ""})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestDeletePolicy_NotFound(t *testing.T) {
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, &fakeAdminStore{}, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.DeletePolicy(bearerCtx("tok"), &adminv1.DeletePolicyRequest{Id: "missing"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestDeletePolicy_Success(t *testing.T) {
	st := &fakeAdminStore{policies: []store.Policy{{ID: "p1", SPIFFEID: "s", Namespace: "ns", Pattern: "ns/svc/*", Permissions: []string{"get"}}}}
	srv := NewAdminServer(&fakeUnsealMgr{}, &fakeTokenChecker{}, st, &fakeKeyUnwrapper{key: adminTestKey})
	_, err := srv.DeletePolicy(bearerCtx("tok"), &adminv1.DeletePolicyRequest{Id: "p1"})
	require.NoError(t, err)
	assert.Empty(t, st.policies)
}
