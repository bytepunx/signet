package auth

import (
	"context"
	"fmt"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// TokenValidator validates Kubernetes ServiceAccount tokens via the TokenReview API.
// It is used exclusively on the admin gRPC endpoint, which is only reachable via
// kubectl port-forward.
type TokenValidator struct {
	client    kubernetes.Interface
	audiences []string // expected token audiences; nil means any audience is accepted
}

// NewTokenValidator creates a TokenValidator backed by the given Kubernetes client.
// audiences restricts which SA tokens are accepted; pass nil to accept any audience.
func NewTokenValidator(client kubernetes.Interface, audiences []string) *TokenValidator {
	return &TokenValidator{client: client, audiences: audiences}
}

// Validate submits token to the Kubernetes TokenReview API. Returns nil if the
// token is valid and authenticated. Returns ErrInvalidToken if the API rejects it.
// Returns ErrUnauthenticated if the token is empty.
func (v *TokenValidator) Validate(ctx context.Context, token string) error {
	if token == "" {
		return fmt.Errorf("%w: token must not be empty", ErrUnauthenticated)
	}

	review := &authv1.TokenReview{
		Spec: authv1.TokenReviewSpec{
			Token:     token,
			Audiences: v.audiences,
		},
	}

	result, err := v.client.AuthenticationV1().TokenReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("auth: token review API: %w", err)
	}

	if !result.Status.Authenticated {
		reason := result.Status.Error
		if reason == "" {
			reason = "token not authenticated"
		}
		return fmt.Errorf("%w: %s", ErrInvalidToken, reason)
	}

	return nil
}

// TokenFromMetadata extracts the bearer token from the gRPC metadata header
// "authorization: Bearer <token>".
func TokenFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadataFromContext(ctx)
	if !ok {
		return "", fmt.Errorf("%w: no metadata in context", ErrUnauthenticated)
	}

	vals := md["authorization"]
	if len(vals) == 0 {
		return "", fmt.Errorf("%w: authorization header missing", ErrUnauthenticated)
	}

	const prefix = "Bearer "
	val := vals[0]
	if len(val) <= len(prefix) || val[:len(prefix)] != prefix {
		return "", fmt.Errorf("%w: authorization header must use Bearer scheme", ErrUnauthenticated)
	}

	token := val[len(prefix):]
	if token == "" {
		return "", fmt.Errorf("%w: bearer token is empty", ErrUnauthenticated)
	}
	return token, nil
}
