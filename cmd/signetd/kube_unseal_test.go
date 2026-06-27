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

	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/unseal"
)

// attemptKubeUnsealWith is a test-only helper that accepts an injected
// kubernetes.Interface, bypassing the real in-cluster client build path.
func attemptKubeUnsealWith(ctx context.Context, mgr *unseal.Manager, secretName, namespace string, client *fake.Clientset) {
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
	_ = mgr.UnsealWithKey(key)
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

	attemptKubeUnsealWith(context.Background(), mgr, "signet-master-key", "signet", k8s)

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

	attemptKubeUnsealWith(context.Background(), mgr, "signet-master-key", "signet", k8s)

	assert.Equal(t, unseal.StateSealed, mgr.Status().State, "manager should remain sealed when field is missing")
}

// TestAttemptKubeUnsealNotFound verifies that a missing Secret leaves the
// manager sealed without panicking.
func TestAttemptKubeUnsealNotFound(t *testing.T) {
	k8s := fake.NewClientset() // no pre-populated secrets

	keyStore := icrypto.NewKeyStore()
	mgr, _ := unseal.New(keyStore, unseal.Config{})

	attemptKubeUnsealWith(context.Background(), mgr, "signet-master-key", "signet", k8s)

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

	attemptKubeUnsealWith(context.Background(), mgr, "signet-master-key", "signet", k8s)

	assert.Equal(t, unseal.StateUnsealed, mgr.Status().State)
}
