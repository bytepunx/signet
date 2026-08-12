package server_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/bytepunx/signet/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// writeSelfSignedCert generates a self-signed ECDSA cert/key pair (serial
// distinguishes successive calls so reload tests can tell certificates
// apart), PEM-encodes both to files under dir, and returns their paths.
func writeSelfSignedCert(t *testing.T, dir string, serial int64) (certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: "signet-admin-test"},
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}

	certFile = filepath.Join(dir, "tls.crt")
	keyFile = filepath.Join(dir, "tls.key")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return certFile, keyFile
}

func TestNew_AdminTLSCertWithoutKey(t *testing.T) {
	dir := t.TempDir()
	certFile, _ := writeSelfSignedCert(t, dir, 1)
	_, err := server.New(
		server.Config{WorkloadAddr: "127.0.0.1:0", AdminAddr: "127.0.0.1:0", AdminTLSCertFile: certFile},
		insecure.NewCredentials(), stubSecrets{}, stubAdmin{}, stubGitOps{}, nil, &fakeMgr{},
	)
	if err == nil {
		t.Fatal("expected error when AdminTLSCertFile is set without AdminTLSKeyFile")
	}
}

func TestNew_AdminTLSKeyWithoutCert(t *testing.T) {
	dir := t.TempDir()
	_, keyFile := writeSelfSignedCert(t, dir, 1)
	_, err := server.New(
		server.Config{WorkloadAddr: "127.0.0.1:0", AdminAddr: "127.0.0.1:0", AdminTLSKeyFile: keyFile},
		insecure.NewCredentials(), stubSecrets{}, stubAdmin{}, stubGitOps{}, nil, &fakeMgr{},
	)
	if err == nil {
		t.Fatal("expected error when AdminTLSKeyFile is set without AdminTLSCertFile")
	}
}

func TestNew_AdminTLSUnreadableFiles(t *testing.T) {
	dir := t.TempDir()
	_, err := server.New(
		server.Config{
			WorkloadAddr:     "127.0.0.1:0",
			AdminAddr:        "127.0.0.1:0",
			AdminTLSCertFile: filepath.Join(dir, "missing.crt"),
			AdminTLSKeyFile:  filepath.Join(dir, "missing.key"),
		},
		insecure.NewCredentials(), stubSecrets{}, stubAdmin{}, stubGitOps{}, nil, &fakeMgr{},
	)
	if err == nil {
		t.Fatal("expected error when the configured cert/key files don't exist")
	}
}

// TestRun_AdminListenerTLS_TerminatesRealTLS is the regression test for
// bytepunx/signet#24: once AdminTLSCertFile/AdminTLSKeyFile are configured,
// the admin listener must actually terminate TLS (a plaintext client must be
// rejected) rather than silently continuing to accept plaintext connections.
func TestRun_AdminListenerTLS_TerminatesRealTLS(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := writeSelfSignedCert(t, dir, 1)

	srv, err := server.New(
		server.Config{
			WorkloadAddr:     "127.0.0.1:0",
			AdminAddr:        "127.0.0.1:0",
			AdminTLSCertFile: certFile,
			AdminTLSKeyFile:  keyFile,
		},
		insecure.NewCredentials(), stubSecrets{}, stubAdmin{}, stubGitOps{}, nil, &fakeMgr{},
	)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	cancel, _ := runBackground(srv)
	defer cancel()
	waitReady()

	// A TLS client (trusting the self-signed cert, as a real deployment would
	// trust its cert-manager-issued CA) must be able to complete RPCs.
	tlsConn, err := grpc.NewClient(srv.AdminAddr().String(),
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	if err != nil {
		t.Fatalf("dial admin over TLS: %v", err)
	}
	defer tlsConn.Close()
	_, err = adminv1.NewAdminServiceClient(tlsConn).Status(context.Background(), &adminv1.StatusRequest{})
	if status.Code(err) != codes.Unimplemented {
		t.Errorf("TLS client: want Unimplemented (proof the RPC reached the handler), got %v", err)
	}

	// A plaintext client must be rejected outright — this is the whole point
	// of #24: a bearer token must never be able to cross the wire in the
	// clear once the admin listener is reachable off loopback.
	plainConn, err := grpc.NewClient(srv.AdminAddr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial admin plaintext: %v", err)
	}
	defer plainConn.Close()
	_, err = adminv1.NewAdminServiceClient(plainConn).Status(context.Background(), &adminv1.StatusRequest{})
	if err == nil {
		t.Fatal("expected a plaintext RPC against a TLS-terminated admin listener to fail")
	}
}

// TestRun_AdminListenerTLS_ReloadsRotatedCertificate verifies a cert-manager
// style rotation (the cert/key files rewritten in place) is picked up
// without a restart, within AdminTLSReloadInterval.
func TestRun_AdminListenerTLS_ReloadsRotatedCertificate(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := writeSelfSignedCert(t, dir, 1)

	srv, err := server.New(
		server.Config{
			WorkloadAddr:           "127.0.0.1:0",
			AdminAddr:              "127.0.0.1:0",
			AdminTLSCertFile:       certFile,
			AdminTLSKeyFile:        keyFile,
			AdminTLSReloadInterval: 20 * time.Millisecond,
		},
		insecure.NewCredentials(), stubSecrets{}, stubAdmin{}, stubGitOps{}, nil, &fakeMgr{},
	)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	cancel, _ := runBackground(srv)
	defer cancel()
	waitReady()

	dialTLS := func() *x509.Certificate {
		t.Helper()
		conn, err := tls.Dial("tcp", srv.AdminAddr().String(), &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			t.Fatalf("tls.Dial: %v", err)
		}
		defer conn.Close()
		certs := conn.ConnectionState().PeerCertificates
		if len(certs) == 0 {
			t.Fatal("no peer certificates presented")
		}
		return certs[0]
	}

	before := dialTLS()
	if before.SerialNumber.Int64() != 1 {
		t.Fatalf("initial cert serial = %d, want 1", before.SerialNumber.Int64())
	}

	// Rewrite the cert/key files in place, as cert-manager's Secret-volume
	// rotation does.
	writeSelfSignedCert(t, dir, 2)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got := dialTLS(); got.SerialNumber.Int64() == 2 {
			return // reload observed
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("admin listener never picked up the rotated certificate")
}
