package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials/insecure"
)

func TestIsLoopbackHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"example.com", false},
		{"10.0.0.5", false},
		{"signet-admin.signet.svc.cluster.local", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			assert.Equal(t, tc.want, isLoopbackHost(tc.host))
		})
	}
}

// TestAdminTransportCreds_LoopbackDefaultsToInsecure is the pre-existing,
// still-supported kubectl port-forward workflow: loopback + no flags stays
// plaintext.
func TestAdminTransportCreds_LoopbackDefaultsToInsecure(t *testing.T) {
	creds, requireTLS, err := adminTransportCreds("localhost:8444", "", false)
	require.NoError(t, err)
	assert.False(t, requireTLS)
	assert.Equal(t, insecure.NewCredentials().Info().SecurityProtocol, creds.Info().SecurityProtocol)
}

func TestAdminTransportCreds_LoopbackIPDefaultsToInsecure(t *testing.T) {
	_, requireTLS, err := adminTransportCreds("127.0.0.1:8444", "", false)
	require.NoError(t, err)
	assert.False(t, requireTLS)
}

// TestAdminTransportCreds_NonLoopbackAlwaysUsesTLS is the H-6 regression
// test: a non-loopback address must never receive insecure credentials, even
// with no flags at all.
func TestAdminTransportCreds_NonLoopbackAlwaysUsesTLS(t *testing.T) {
	creds, requireTLS, err := adminTransportCreds("signet-admin.example.com:8444", "", false)
	require.NoError(t, err)
	assert.True(t, requireTLS)
	assert.NotEqual(t, "insecure", creds.Info().SecurityProtocol)
}

func TestAdminTransportCreds_ForceTLSOnLoopback(t *testing.T) {
	creds, requireTLS, err := adminTransportCreds("localhost:8444", "", true)
	require.NoError(t, err)
	assert.True(t, requireTLS)
	assert.NotEqual(t, "insecure", creds.Info().SecurityProtocol)
}

func TestAdminTransportCreds_CustomCALoadedForNonLoopback(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	require.NoError(t, os.WriteFile(caPath, []byte(testCAPEM), 0o600))

	_, requireTLS, err := adminTransportCreds("signet-admin.example.com:8444", caPath, false)
	require.NoError(t, err)
	assert.True(t, requireTLS)
}

func TestAdminTransportCreds_MissingCAFileErrors(t *testing.T) {
	_, _, err := adminTransportCreds("signet-admin.example.com:8444", "/nonexistent/ca.pem", false)
	require.Error(t, err)
}

func TestAdminTransportCreds_InvalidCAPEMErrors(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	require.NoError(t, os.WriteFile(caPath, []byte("not a real cert"), 0o600))

	_, _, err := adminTransportCreds("signet-admin.example.com:8444", caPath, false)
	require.Error(t, err)
}

func TestTokenCreds_RequireTransportSecurityReflectsTLSState(t *testing.T) {
	assert.False(t, tokenCreds{token: "t", requireTLS: false}.RequireTransportSecurity())
	assert.True(t, tokenCreds{token: "t", requireTLS: true}.RequireTransportSecurity())
}

func TestTokenCreds_InjectsBearerHeader(t *testing.T) {
	md, err := tokenCreds{token: "abc123"}.GetRequestMetadata(nil) //nolint:staticcheck // nil ctx is fine here
	require.NoError(t, err)
	assert.Equal(t, "Bearer abc123", md["authorization"])
}

// testCAPEM is a real self-signed certificate (P-256, CN=test-ca, generated
// via crypto/x509) used only to exercise AppendCertsFromPEM's success path.
const testCAPEM = `-----BEGIN CERTIFICATE-----
MIIBVDCB+6ADAgECAgEBMAoGCCqGSM49BAMCMBIxEDAOBgNVBAMTB3Rlc3QtY2Ew
HhcNMjMxMTE0MjIxMzIwWhcNMzMxMTExMjIxMzIwWjASMRAwDgYDVQQDEwd0ZXN0
LWNhMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEqpR6rVlFsX1plhVeqONCz0Y5
tWpYuMGfKpXNxo+GXj6sjaQozzIQRHpeSeT4FcFmLd0HX5hXL7xXyedG6r8N3qNC
MEAwDgYDVR0PAQH/BAQDAgKEMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFMEk
KxevOyU8ZLi0lgOqD8ABJm0TMAoGCCqGSM49BAMCA0gAMEUCIDpe5M7NVhRnvQEt
Q3q9C52tJ4xwrMwbYHuohmRPzWcWAiEA4PmhZ8QWTbeG5qCkNQ2ksxFmbH7F8bvX
pyzMCAeuuJ8=
-----END CERTIFICATE-----`
