package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"google.golang.org/grpc"
)

// mockAdminClient implements adminv1.AdminServiceClient for testing.
// statusStates controls what Status returns on each successive call.
type mockAdminClient struct {
	statusStates   []adminv1.StatusResponse_State
	statusIdx      int
	unsealKeyErr   error
	unsealKeyCalls [][]byte // a copy of each key received
}

func (m *mockAdminClient) Status(_ context.Context, _ *adminv1.StatusRequest, _ ...grpc.CallOption) (*adminv1.StatusResponse, error) {
	if m.statusIdx < len(m.statusStates) {
		s := m.statusStates[m.statusIdx]
		m.statusIdx++
		return &adminv1.StatusResponse{State: s}, nil
	}
	return &adminv1.StatusResponse{State: adminv1.StatusResponse_STATE_UNSEALED}, nil
}

func (m *mockAdminClient) UnsealKey(_ context.Context, in *adminv1.UnsealKeyRequest, _ ...grpc.CallOption) (*adminv1.UnsealKeyResponse, error) {
	cp := make([]byte, len(in.Key))
	copy(cp, in.Key)
	m.unsealKeyCalls = append(m.unsealKeyCalls, cp)
	return &adminv1.UnsealKeyResponse{}, m.unsealKeyErr
}

func (m *mockAdminClient) UnsealShare(_ context.Context, _ *adminv1.UnsealShareRequest, _ ...grpc.CallOption) (*adminv1.UnsealShareResponse, error) {
	return &adminv1.UnsealShareResponse{}, nil
}

func (m *mockAdminClient) Seal(_ context.Context, _ *adminv1.SealRequest, _ ...grpc.CallOption) (*adminv1.SealResponse, error) {
	return &adminv1.SealResponse{}, nil
}

const testNS = "signet"
const testSecret = "signet-master-key"

// callInit runs runInitWithDeps with the given mock admin client and fake k8s
// client; returns stdout output.
func callInit(t *testing.T, admin *mockAdminClient, k8s *fake.Clientset, force, dryRun bool) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := runInitWithDeps(context.Background(), &out, admin, k8s,
		testNS, testSecret, force, dryRun)
	return out.String(), err
}

// TestInitCreatePath verifies that when no Secret exists a new 32-byte key is
// generated, the Secret is created, and UnsealKey is called with that key.
func TestInitCreatePath(t *testing.T) {
	admin := &mockAdminClient{statusStates: []adminv1.StatusResponse_State{
		adminv1.StatusResponse_STATE_SEALED,
		adminv1.StatusResponse_STATE_UNSEALED,
	}}
	k8s := fake.NewClientset()

	out, err := callInit(t, admin, k8s, false, false)
	require.NoError(t, err)

	// Secret must have been created.
	sec, getErr := k8s.CoreV1().Secrets(testNS).Get(context.Background(), testSecret, metav1.GetOptions{})
	require.NoError(t, getErr, "Secret should have been created")
	require.Contains(t, sec.Data, "master.key", "Secret must contain master.key field")
	assert.Len(t, sec.Data["master.key"], 32, "master.key must be 32 bytes")

	// UnsealKey called exactly once with the key that ended up in the Secret.
	require.Len(t, admin.unsealKeyCalls, 1, "UnsealKey must be called once")
	assert.Equal(t, sec.Data["master.key"], admin.unsealKeyCalls[0],
		"UnsealKey must receive the key written to the Secret")

	assert.Contains(t, out, "Created Secret")
	assert.Contains(t, out, "WARNING")
}

// TestInitReadPath verifies that when a valid Secret already exists the
// existing key is used without regeneration.
func TestInitReadPath(t *testing.T) {
	existingKey := bytes.Repeat([]byte{0xAB}, 32)
	preExisting := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testSecret, Namespace: testNS},
		Data:       map[string][]byte{"master.key": existingKey},
	}
	admin := &mockAdminClient{statusStates: []adminv1.StatusResponse_State{
		adminv1.StatusResponse_STATE_SEALED,
		adminv1.StatusResponse_STATE_UNSEALED,
	}}
	k8s := fake.NewClientset(preExisting)

	out, err := callInit(t, admin, k8s, false, false)
	require.NoError(t, err)

	// UnsealKey called with the original key.
	require.Len(t, admin.unsealKeyCalls, 1)
	assert.Equal(t, existingKey, admin.unsealKeyCalls[0])

	assert.Contains(t, out, "existing master key")
	assert.NotContains(t, out, "WARNING")
}

// TestInitForce verifies that --force regenerates the key and overwrites the
// existing Secret.
func TestInitForce(t *testing.T) {
	oldKey := bytes.Repeat([]byte{0x01}, 32)
	preExisting := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testSecret, Namespace: testNS},
		Data:       map[string][]byte{"master.key": oldKey},
	}
	admin := &mockAdminClient{statusStates: []adminv1.StatusResponse_State{
		adminv1.StatusResponse_STATE_SEALED,
		adminv1.StatusResponse_STATE_UNSEALED,
	}}
	k8s := fake.NewClientset(preExisting)

	_, err := callInit(t, admin, k8s, true, false)
	require.NoError(t, err)

	// Secret must have been updated with a different key.
	sec, _ := k8s.CoreV1().Secrets(testNS).Get(context.Background(), testSecret, metav1.GetOptions{})
	newKey := sec.Data["master.key"]
	assert.Len(t, newKey, 32)
	assert.NotEqual(t, oldKey, newKey, "key must be regenerated under --force")

	// UnsealKey called with the new key, not the old one.
	require.Len(t, admin.unsealKeyCalls, 1)
	assert.NotEqual(t, oldKey, admin.unsealKeyCalls[0])
	assert.Equal(t, newKey, admin.unsealKeyCalls[0])
}

// TestInitAlreadyUnsealed verifies that the command exits 0 without touching
// the Secret or calling UnsealKey when signet is already unsealed.
func TestInitAlreadyUnsealed(t *testing.T) {
	admin := &mockAdminClient{statusStates: []adminv1.StatusResponse_State{
		adminv1.StatusResponse_STATE_UNSEALED,
	}}
	k8s := fake.NewClientset()

	out, err := callInit(t, admin, k8s, false, false)
	require.NoError(t, err)

	// No gRPC or Kubernetes writes.
	assert.Len(t, admin.unsealKeyCalls, 0, "UnsealKey must not be called")
	secrets, _ := k8s.CoreV1().Secrets(testNS).List(context.Background(), metav1.ListOptions{})
	assert.Len(t, secrets.Items, 0, "no Secret must be created")

	assert.True(t, strings.Contains(out, "already unsealed"))
}

// TestInitDryRun verifies that --dry-run makes no writes and no gRPC calls
// beyond Status.
func TestInitDryRun(t *testing.T) {
	admin := &mockAdminClient{statusStates: []adminv1.StatusResponse_State{
		adminv1.StatusResponse_STATE_SEALED,
	}}
	k8s := fake.NewClientset()

	out, err := callInit(t, admin, k8s, false, true)
	require.NoError(t, err)

	// No Secret created.
	secrets, _ := k8s.CoreV1().Secrets(testNS).List(context.Background(), metav1.ListOptions{})
	assert.Len(t, secrets.Items, 0, "no Secret must be created in dry-run")

	// No UnsealKey call.
	assert.Len(t, admin.unsealKeyCalls, 0, "UnsealKey must not be called in dry-run")

	assert.Contains(t, out, "[dry-run]")
	assert.Contains(t, out, "no changes made")
}
