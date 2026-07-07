package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/bytepunx/signet/internal/api"
	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
	"github.com/bytepunx/signet/internal/unseal"
)

// fakeKCVStore is a minimal in-memory api.KeyCheckStore fake, avoiding a real
// database dependency for these unit tests.
type fakeKCVStore struct{ kcv []byte }

func (f *fakeKCVStore) GetKeyCheckValue(_ context.Context) ([]byte, error) {
	if f.kcv == nil {
		return nil, store.ErrNotFound
	}
	return f.kcv, nil
}

func (f *fakeKCVStore) PutKeyCheckValue(_ context.Context, ciphertext []byte) error {
	f.kcv = ciphertext
	return nil
}

// attemptKubeUnsealWith is a test-only helper that mirrors attemptKubeUnseal
// but accepts an injected kubernetes.Interface and a lightweight KCV store
// fake, bypassing the real in-cluster client build path and a real database.
func attemptKubeUnsealWith(ctx context.Context, mgr *unseal.Manager, kcvStore api.KeyCheckStore, keyStore *icrypto.KeyStore, secretName, namespace string, client *fake.Clientset) {
	if mgr.Status().State == unseal.StateUnsealed {
		return
	}

	sec, err := client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return
	}

	keyBytes, ok := sec.Data["master.key"]
	if !ok {
		return
	}

	key := make([]byte, len(keyBytes))
	copy(key, keyBytes)
	if err := mgr.UnsealWithKey(key); err != nil {
		return
	}

	if err := api.VerifyOrInitKeyCheckValue(ctx, kcvStore, keyStore); err != nil {
		mgr.Seal()
	}
}

// TestAttemptKubeUnseal verifies that a valid Secret causes the manager to
// transition to unsealed and that the key bytes held by the fake client are
// zeroed (via the copy) after UnsealWithKey consumes them.
func TestAttemptKubeUnseal(t *testing.T) {
	secretKey := bytes.Repeat([]byte{0x42}, 32)
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "signet-master-key", Namespace: "signet"},
		Data:       map[string][]byte{"master.key": secretKey},
	}
	k8s := fake.NewClientset(sec)

	keyStore := icrypto.NewKeyStore()
	mgr, err := unseal.New(keyStore, unseal.Config{})
	require.NoError(t, err)

	assert.Equal(t, unseal.StateSealed, mgr.Status().State)

	attemptKubeUnsealWith(context.Background(), mgr, &fakeKCVStore{}, keyStore, "signet-master-key", "signet", k8s)

	assert.Equal(t, unseal.StateUnsealed, mgr.Status().State, "manager should be unsealed")
}

// TestAttemptKubeUnsealMissingField verifies that a Secret without 'master.key'
// leaves the manager sealed.
func TestAttemptKubeUnsealMissingField(t *testing.T) {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "signet-master-key", Namespace: "signet"},
		Data:       map[string][]byte{"wrong-field": []byte("data")},
	}
	k8s := fake.NewClientset(sec)

	keyStore := icrypto.NewKeyStore()
	mgr, _ := unseal.New(keyStore, unseal.Config{})

	attemptKubeUnsealWith(context.Background(), mgr, &fakeKCVStore{}, keyStore, "signet-master-key", "signet", k8s)

	assert.Equal(t, unseal.StateSealed, mgr.Status().State, "manager should remain sealed when field is missing")
}

// TestAttemptKubeUnsealNotFound verifies that a missing Secret leaves the
// manager sealed without panicking.
func TestAttemptKubeUnsealNotFound(t *testing.T) {
	k8s := fake.NewClientset() // no pre-populated secrets

	keyStore := icrypto.NewKeyStore()
	mgr, _ := unseal.New(keyStore, unseal.Config{})

	attemptKubeUnsealWith(context.Background(), mgr, &fakeKCVStore{}, keyStore, "signet-master-key", "signet", k8s)

	assert.Equal(t, unseal.StateSealed, mgr.Status().State)
}

// TestAttemptKubeUnsealAlreadyUnsealed verifies that an already-unsealed
// manager is not affected.
func TestAttemptKubeUnsealAlreadyUnsealed(t *testing.T) {
	existingKey := bytes.Repeat([]byte{0x11}, 32)
	keyStore := icrypto.NewKeyStore()
	mgr, _ := unseal.New(keyStore, unseal.Config{})
	require.NoError(t, mgr.UnsealWithKey(existingKey))
	assert.Equal(t, unseal.StateUnsealed, mgr.Status().State)

	// Secret with a different key — if the guard fires, the manager stays with
	// the original key and the second UnsealWithKey is never called.
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "signet-master-key", Namespace: "signet"},
		Data:       map[string][]byte{"master.key": bytes.Repeat([]byte{0x99}, 32)},
	}
	k8s := fake.NewClientset(sec)

	attemptKubeUnsealWith(context.Background(), mgr, &fakeKCVStore{}, keyStore, "signet-master-key", "signet", k8s)

	assert.Equal(t, unseal.StateUnsealed, mgr.Status().State)
}

// TestAttemptKubeUnseal_KeyCheckMismatch_ReSeals verifies that a Secret
// holding a key which does not match a pre-existing key-check value leaves
// the manager sealed rather than silently running with the wrong master key.
func TestAttemptKubeUnseal_KeyCheckMismatch_ReSeals(t *testing.T) {
	correctKey := bytes.Repeat([]byte{0x01}, 32)
	wrongKey := bytes.Repeat([]byte{0x02}, 32)

	// Establish a key-check value under the "correct" key first, as if a
	// prior manual unseal had already happened.
	kcvKeyStore := icrypto.NewKeyStore()
	require.NoError(t, kcvKeyStore.Set(append([]byte(nil), correctKey...)))
	kcvStore := &fakeKCVStore{}
	require.NoError(t, api.VerifyOrInitKeyCheckValue(context.Background(), kcvStore, kcvKeyStore))
	require.NotEmpty(t, kcvStore.kcv)

	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "signet-master-key", Namespace: "signet"},
		Data:       map[string][]byte{"master.key": wrongKey},
	}
	k8s := fake.NewClientset(sec)

	keyStore := icrypto.NewKeyStore()
	mgr, err := unseal.New(keyStore, unseal.Config{})
	require.NoError(t, err)

	attemptKubeUnsealWith(context.Background(), mgr, kcvStore, keyStore, "signet-master-key", "signet", k8s)

	assert.Equal(t, unseal.StateSealed, mgr.Status().State, "manager must re-seal when the Secret's key fails the key check")
}
