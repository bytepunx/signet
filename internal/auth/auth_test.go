package auth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// --- certificate helpers ---

func selfSignedCertWithURIs(t *testing.T, uris ...*url.URL) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		URIs:         uris,
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

func spiffeURI(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}

func ctxWithVerifiedCert(t *testing.T, cert *x509.Certificate) context.Context {
	t.Helper()
	state := tls.ConnectionState{
		VerifiedChains: [][]*x509.Certificate{{cert}},
	}
	p := &peer.Peer{AuthInfo: credentials.TLSInfo{State: state}}
	return peer.NewContext(context.Background(), p)
}

func ctxWithMD(kv map[string]string) context.Context {
	md := make(metadata.MD, len(kv))
	for k, v := range kv {
		md[k] = []string{v}
	}
	return metadata.NewIncomingContext(context.Background(), md)
}

// --- SPIFFEIDFromContext ---

func TestSPIFFEIDFromContext_NoPeer(t *testing.T) {
	_, err := SPIFFEIDFromContext(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthenticated)
}

func TestSPIFFEIDFromContext_NotTLS(t *testing.T) {
	p := &peer.Peer{AuthInfo: nil}
	ctx := peer.NewContext(context.Background(), p)
	_, err := SPIFFEIDFromContext(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthenticated)
}

func TestSPIFFEIDFromContext_NoVerifiedChain(t *testing.T) {
	state := tls.ConnectionState{} // empty VerifiedChains
	p := &peer.Peer{AuthInfo: credentials.TLSInfo{State: state}}
	ctx := peer.NewContext(context.Background(), p)
	_, err := SPIFFEIDFromContext(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthenticated)
}

func TestSPIFFEIDFromContext_NoSPIFFEURI(t *testing.T) {
	cert := selfSignedCertWithURIs(t) // no URIs
	ctx := ctxWithVerifiedCert(t, cert)
	_, err := SPIFFEIDFromContext(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthenticated)
	assert.Contains(t, err.Error(), "no SPIFFE URI SAN")
}

func TestSPIFFEIDFromContext_NonSPIFFEURI(t *testing.T) {
	cert := selfSignedCertWithURIs(t, spiffeURI("https://example.com/service"))
	ctx := ctxWithVerifiedCert(t, cert)
	_, err := SPIFFEIDFromContext(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthenticated)
}

func TestSPIFFEIDFromContext_ValidSPIFFEID(t *testing.T) {
	spiffeID := "spiffe://cluster.local/ns/prod/sa/api"
	cert := selfSignedCertWithURIs(t, spiffeURI(spiffeID))
	ctx := ctxWithVerifiedCert(t, cert)
	got, err := SPIFFEIDFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, spiffeID, got)
}

func TestSPIFFEIDFromContext_MultipleSPIFFEIDs(t *testing.T) {
	cert := selfSignedCertWithURIs(t,
		spiffeURI("spiffe://cluster.local/ns/prod/sa/api"),
		spiffeURI("spiffe://cluster.local/ns/prod/sa/worker"),
	)
	ctx := ctxWithVerifiedCert(t, cert)
	_, err := SPIFFEIDFromContext(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthenticated)
	assert.Contains(t, err.Error(), "2 SPIFFE URI SANs")
}

func TestSPIFFEIDFromContext_MixedURIs(t *testing.T) {
	// One non-SPIFFE URI alongside one SPIFFE URI: must extract just the SPIFFE one.
	spiffeID := "spiffe://cluster.local/ns/prod/sa/api"
	cert := selfSignedCertWithURIs(t,
		spiffeURI("https://example.com/service"),
		spiffeURI(spiffeID),
	)
	ctx := ctxWithVerifiedCert(t, cert)
	got, err := SPIFFEIDFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, spiffeID, got)
}

// TestSPIFFEIDFromContext_RealMTLSHandshake exercises the actual transport
// credentials internal/server uses in production
// (grpccredentials.MTLSServerCredentials/MTLSClientCredentials) instead of
// hand-constructing a peer.Peer{AuthInfo: credentials.TLSInfo{...}} like the
// tests above. Those hand-built contexts don't match what a real go-spiffe
// mTLS handshake produces (AuthInfo comes back wrapped in an unexported
// type), which is exactly how the "connection is not mTLS" regression
// shipped undetected — every unit test authenticated fine while every real
// workload connection failed. This test would have caught it.
func TestSPIFFEIDFromContext_RealMTLSHandshake(t *testing.T) {
	td := spiffeid.RequireTrustDomainFromString("test.example.org")

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	issue := func(id spiffeid.ID) (*x509.Certificate, crypto.Signer) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(2),
			Subject:      pkix.Name{CommonName: id.String()},
			URIs:         []*url.URL{id.URL()},
			NotBefore:    time.Now().Add(-time.Minute),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		}
		der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
		require.NoError(t, err)
		cert, err := x509.ParseCertificate(der)
		require.NoError(t, err)
		return cert, key
	}

	serverID := spiffeid.RequireFromString("spiffe://test.example.org/ns/signet/sa/signetd")
	clientID := spiffeid.RequireFromString("spiffe://test.example.org/ns/prod/sa/api")
	serverCert, serverKey := issue(serverID)
	clientCert, clientKey := issue(clientID)

	bundle := x509bundle.FromX509Authorities(td, []*x509.Certificate{caCert})
	serverSVID := &x509svid.SVID{ID: serverID, Certificates: []*x509.Certificate{serverCert}, PrivateKey: serverKey}
	clientSVID := &x509svid.SVID{ID: clientID, Certificates: []*x509.Certificate{clientCert}, PrivateKey: clientKey}

	serverCreds := grpccredentials.MTLSServerCredentials(serverSVID, bundle, tlsconfig.AuthorizeMemberOf(td))
	clientCreds := grpccredentials.MTLSClientCredentials(clientSVID, bundle, tlsconfig.AuthorizeMemberOf(td))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	authInfoCh := make(chan credentials.AuthInfo, 1)
	serverErrCh := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			serverErrCh <- err
			return
		}
		_, authInfo, err := serverCreds.ServerHandshake(conn)
		if err != nil {
			serverErrCh <- err
			return
		}
		authInfoCh <- authInfo
		serverErrCh <- nil
	}()

	clientConn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	defer clientConn.Close()
	_, _, err = clientCreds.ClientHandshake(context.Background(), "", clientConn)
	require.NoError(t, err)
	require.NoError(t, <-serverErrCh)

	p := &peer.Peer{AuthInfo: <-authInfoCh}
	ctx := peer.NewContext(context.Background(), p)

	got, err := SPIFFEIDFromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, clientID.String(), got)
}

// --- spiffeURIFromCert ---

func TestSpiffeURIFromCert_Empty(t *testing.T) {
	_, err := spiffeURIFromCert(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no SPIFFE URI SAN")
}

func TestSpiffeURIFromCert_OneValid(t *testing.T) {
	u := spiffeURI("spiffe://example.org/svc")
	got, err := spiffeURIFromCert([]*url.URL{u})
	require.NoError(t, err)
	assert.Equal(t, "spiffe://example.org/svc", got)
}

func TestSpiffeURIFromCert_TwoSPIFFE(t *testing.T) {
	_, err := spiffeURIFromCert([]*url.URL{
		spiffeURI("spiffe://example.org/a"),
		spiffeURI("spiffe://example.org/b"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2 SPIFFE URI SANs")
}

func TestSpiffeURIFromCert_OnlyNonSPIFFE(t *testing.T) {
	_, err := spiffeURIFromCert([]*url.URL{spiffeURI("https://example.org")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no SPIFFE URI SAN")
}

// --- TokenFromMetadata ---

func TestTokenFromMetadata_NoMetadata(t *testing.T) {
	_, err := TokenFromMetadata(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthenticated)
}

func TestTokenFromMetadata_MissingAuthHeader(t *testing.T) {
	ctx := ctxWithMD(map[string]string{})
	_, err := TokenFromMetadata(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthenticated)
	assert.Contains(t, err.Error(), "authorization header missing")
}

func TestTokenFromMetadata_WrongScheme(t *testing.T) {
	ctx := ctxWithMD(map[string]string{"authorization": "Basic dXNlcjpwYXNz"})
	_, err := TokenFromMetadata(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthenticated)
	assert.Contains(t, err.Error(), "Bearer scheme")
}

func TestTokenFromMetadata_BearerPrefixOnly(t *testing.T) {
	ctx := ctxWithMD(map[string]string{"authorization": "Bearer "})
	_, err := TokenFromMetadata(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthenticated)
}

func TestTokenFromMetadata_ValidToken(t *testing.T) {
	ctx := ctxWithMD(map[string]string{"authorization": "Bearer mytoken123"})
	token, err := TokenFromMetadata(ctx)
	require.NoError(t, err)
	assert.Equal(t, "mytoken123", token)
}

func TestTokenFromMetadata_JWTToken(t *testing.T) {
	jwt := "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJzYSJ9.sig"
	ctx := ctxWithMD(map[string]string{"authorization": "Bearer " + jwt})
	token, err := TokenFromMetadata(ctx)
	require.NoError(t, err)
	assert.Equal(t, jwt, token)
}

// --- parseKubeSpiffeID ---

func TestParseKubeSpiffeID(t *testing.T) {
	const configuredTrustDomain = "cluster.local"
	cases := []struct {
		name          string
		spiffeID      string
		wantNamespace string
		wantSA        string
	}{
		{
			name:          "valid kubernetes SVID",
			spiffeID:      "spiffe://cluster.local/ns/payments/sa/api",
			wantNamespace: "payments",
			wantSA:        "api",
		},
		{
			name:     "extra path segments — not kubernetes convention",
			spiffeID: "spiffe://cluster.local/ns/payments/sa/api/extra",
		},
		{
			name:     "too few segments",
			spiffeID: "spiffe://cluster.local/ns/payments",
		},
		{
			name:     "wrong segment labels",
			spiffeID: "spiffe://cluster.local/namespace/payments/serviceaccount/api",
		},
		{
			name:     "empty namespace segment",
			spiffeID: "spiffe://cluster.local/ns//sa/api",
		},
		{
			name:     "empty SA segment",
			spiffeID: "spiffe://cluster.local/ns/payments/sa/",
		},
		{
			name:     "non-SPIFFE scheme",
			spiffeID: "https://cluster.local/ns/payments/sa/api",
		},
		{
			name:     "arbitrary service path",
			spiffeID: "spiffe://cluster.local/service/my-service",
		},
		{
			name:     "empty string",
			spiffeID: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotNS, gotSA := parseKubeSpiffeID(tc.spiffeID, configuredTrustDomain)
			assert.Equal(t, tc.wantNamespace, gotNS)
			assert.Equal(t, tc.wantSA, gotSA)
		})
	}
}

// TestParseKubeSpiffeID_TrustDomainMismatch is the L-9 regression test: an
// otherwise well-formed Kubernetes-convention SPIFFE ID from a DIFFERENT
// trust domain than the one this checker is configured for must not be
// treated as a match, even though the TLS layer (SpireCredentials with
// AuthorizeMemberOf) is expected to have already rejected such a connection —
// this keeps the authorization decision correct on its own.
func TestParseKubeSpiffeID_TrustDomainMismatch(t *testing.T) {
	gotNS, gotSA := parseKubeSpiffeID("spiffe://prod.example.com/ns/infra/sa/worker", "cluster.local")
	assert.Empty(t, gotNS)
	assert.Empty(t, gotSA)
}

func TestParseKubeSpiffeID_TrustDomainMatch(t *testing.T) {
	gotNS, gotSA := parseKubeSpiffeID("spiffe://prod.example.com/ns/infra/sa/worker", "prod.example.com")
	assert.Equal(t, "infra", gotNS)
	assert.Equal(t, "worker", gotSA)
}

// --- sentinel error identity ---

func TestErrors_Distinct(t *testing.T) {
	assert.NotEqual(t, ErrUnauthenticated, ErrUnauthorized)
	assert.NotEqual(t, ErrUnauthenticated, ErrInvalidToken)
	assert.NotEqual(t, ErrUnauthorized, ErrInvalidToken)
}

func TestErrors_IsWrappable(t *testing.T) {
	wrapped := errors.Join(ErrUnauthorized, errors.New("extra"))
	assert.ErrorIs(t, wrapped, ErrUnauthorized)

	wrapped2 := errors.Join(ErrUnauthenticated, errors.New("detail"))
	assert.ErrorIs(t, wrapped2, ErrUnauthenticated)
}
