// Package auth handles SPIFFE ID extraction from verified mTLS connections,
// policy evaluation, and Kubernetes SA token validation for the admin endpoint.
package auth

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/bytepunx/signet/internal/store"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

// ErrUnauthenticated is returned when a caller presents no valid identity.
var ErrUnauthenticated = errors.New("unauthenticated")

// ErrUnauthorized is returned when a caller's identity is known but they lack
// permission for the requested operation.
var ErrUnauthorized = errors.New("unauthorized")

// ErrInvalidToken is returned when an SA token fails Kubernetes TokenReview.
var ErrInvalidToken = errors.New("invalid token")

// --- SPIFFE ID extraction ---

// SPIFFEIDFromContext extracts and validates the SPIFFE ID from the verified
// mTLS peer certificate in the gRPC context. It returns ErrUnauthenticated if
// no peer is present, no TLS info is available, or the certificate carries no
// SPIFFE URI SAN.
//
// The returned SPIFFE ID is already validated by the SPIRE CA certificate chain
// (verification happens in the TLS handshake); this function only parses the
// URI out of the verified cert.
func SPIFFEIDFromContext(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("%w: no peer in context", ErrUnauthenticated)
	}

	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", fmt.Errorf("%w: connection is not mTLS", ErrUnauthenticated)
	}

	state := tlsInfo.State
	if len(state.VerifiedChains) == 0 || len(state.VerifiedChains[0]) == 0 {
		return "", fmt.Errorf("%w: no verified certificate chain", ErrUnauthenticated)
	}

	// The leaf certificate is the first in the first verified chain.
	leaf := state.VerifiedChains[0][0]
	spiffeID, err := spiffeURIFromCert(leaf.URIs)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnauthenticated, err)
	}
	return spiffeID, nil
}

// spiffeURIFromCert finds and validates a spiffe:// URI SAN in the certificate's
// URI list. Returns an error if none is found or more than one is present
// (RFC 8705 §2 prohibits multiple SPIFFE IDs in one certificate).
func spiffeURIFromCert(uris []*url.URL) (string, error) {
	var found []string
	for _, u := range uris {
		if u.Scheme == "spiffe" {
			found = append(found, u.String())
		}
	}
	switch len(found) {
	case 0:
		return "", errors.New("certificate contains no SPIFFE URI SAN")
	case 1:
		return found[0], nil
	default:
		return "", fmt.Errorf("certificate contains %d SPIFFE URI SANs; exactly one is required", len(found))
	}
}

// --- Policy evaluation ---

// Checker evaluates access policies fetched from the store.
type Checker struct {
	st *store.Store
}

// NewChecker creates a Checker backed by the given store.
func NewChecker(st *store.Store) *Checker {
	return &Checker{st: st}
}

// Allow returns nil if the caller is permitted to perform the operation.
//
// Exact-match convention (no policy required): when the SPIFFE ID encodes a
// Kubernetes workload identity of the form
// spiffe://<trust-domain>/ns/<namespace>/sa/<service>, and the encoded
// namespace and service account name exactly match the requested secret's
// namespace and service, access is granted without consulting the policy store.
// This covers the primary use-case — a service reading its own secrets — while
// keeping explicit policies mandatory for every cross-service or wildcard
// access pattern.
//
// All other cases require an explicit policy. Pattern matching uses path.Match
// semantics on the "namespace/secretName" target:
//   - '*' matches any sequence of non-separator characters
//   - '?' matches any single non-separator character
//   - '[abc]' matches character classes
func (c *Checker) Allow(ctx context.Context, spiffeID, permission, namespace, service, secretName string) error {
	if spiffeID == "" {
		return fmt.Errorf("%w: empty SPIFFE ID", ErrUnauthenticated)
	}

	// Exact-match bypass: no policy needed when the workload's own
	// Kubernetes namespace and service account name match the secret's
	// namespace and service exactly.
	if spiffeNS, spiffeSA := parseKubeSpiffeID(spiffeID); spiffeNS != "" &&
		spiffeNS == namespace && spiffeSA == service {
		return nil
	}

	policies, err := c.st.GetPoliciesForSPIFFE(ctx, spiffeID)
	if err != nil {
		return fmt.Errorf("auth: fetch policies: %w", err)
	}

	return evalPolicies(policies, spiffeID, permission, namespace, secretName)
}

// parseKubeSpiffeID extracts the Kubernetes namespace and service account name
// from a SPIRE workload attestor SPIFFE ID of the form:
//
//	spiffe://<trust-domain>/ns/<namespace>/sa/<service-account>
//
// Returns ("", "") for any ID that does not follow this exact convention.
func parseKubeSpiffeID(spiffeID string) (namespace, serviceAccount string) {
	u, err := url.Parse(spiffeID)
	if err != nil || u.Scheme != "spiffe" {
		return "", ""
	}
	// Expect exactly /ns/<namespace>/sa/<service-account> — four non-empty segments.
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "ns" || parts[2] != "sa" ||
		parts[1] == "" || parts[3] == "" {
		return "", ""
	}
	return parts[1], parts[3]
}

// evalPolicies is the pure policy-matching core of Allow, extracted so tests
// can supply a slice of policies directly without a database.
func evalPolicies(policies []store.Policy, spiffeID, permission, namespace, secretName string) error {
	target := namespace + "/" + secretName

	for _, p := range policies {
		if p.Namespace != namespace && p.Namespace != "*" {
			continue
		}
		matched, err := path.Match(p.Pattern, target)
		if err != nil {
			// path.Match only errors on malformed patterns; skip broken policies.
			continue
		}
		if !matched {
			continue
		}
		for _, perm := range p.Permissions {
			if perm == permission || perm == "*" {
				return nil
			}
		}
	}

	return fmt.Errorf("%w: %s does not have %q on %s/%s",
		ErrUnauthorized, spiffeID, permission, namespace, secretName)
}

// --- TLS state extraction (for testing) ---

// tlsStateFromContext is a helper used in tests to inject a fake TLS state.
type tlsStateKey struct{}

// contextWithTLSState embeds a fake TLS state into the context for testing.
func contextWithTLSState(ctx context.Context, state tls.ConnectionState) context.Context {
	return context.WithValue(ctx, tlsStateKey{}, state)
}
