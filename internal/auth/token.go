package auth

import (
	"context"
	"fmt"
	"strings"

	authv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Synthetic resource checked via SubjectAccessReview to authorize admin
// access. It does not correspond to a real Kubernetes API resource — RBAC
// does not require one to exist, only that a ClusterRole/Role grants this
// (group, resource, verb) tuple to the caller. Example:
//
//	apiVersion: rbac.authorization.k8s.io/v1
//	kind: ClusterRole
//	metadata:
//	  name: signet-admin-operator
//	rules:
//	  - apiGroups: ["signet.io"]
//	    resources: ["adminoperations"]
//	    verbs: ["administer"]
const (
	adminSARGroup    = "signet.io"
	adminSARResource = "adminoperations"
	adminSARVerb     = "administer"
)

// TokenValidator validates Kubernetes ServiceAccount tokens via the TokenReview
// API and authorizes the resulting identity for admin access via two
// complementary mechanisms — either is sufficient:
//  1. An explicit allowlist of ServiceAccounts/groups (SIGNET_ADMIN_SUBJECTS).
//  2. A SubjectAccessReview against a synthetic admin resource, delegating
//     the decision to cluster RBAC.
//
// It is used exclusively on the admin gRPC endpoint, which is only reachable
// via kubectl port-forward.
type TokenValidator struct {
	client        kubernetes.Interface
	audiences     []string // expected token audiences; nil means any audience is accepted
	allowedUsers  map[string]bool
	allowedGroups map[string]bool
}

// NewTokenValidator creates a TokenValidator backed by the given Kubernetes
// client. audiences restricts which SA tokens are accepted; pass nil to
// accept any audience. allowedSubjects is a comma-free slice of entries, each
// either "serviceaccount:<namespace>:<name>" or "group:<name>"; pass nil or
// empty to rely solely on the SubjectAccessReview check.
func NewTokenValidator(client kubernetes.Interface, audiences []string, allowedSubjects []string) *TokenValidator {
	users := make(map[string]bool)
	groups := make(map[string]bool)
	for _, s := range allowedSubjects {
		switch {
		case strings.HasPrefix(s, "serviceaccount:"):
			parts := strings.SplitN(strings.TrimPrefix(s, "serviceaccount:"), ":", 2)
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				users[fmt.Sprintf("system:serviceaccount:%s:%s", parts[0], parts[1])] = true
			}
		case strings.HasPrefix(s, "group:"):
			if g := strings.TrimPrefix(s, "group:"); g != "" {
				groups[g] = true
			}
		}
	}
	return &TokenValidator{
		client:        client,
		audiences:     audiences,
		allowedUsers:  users,
		allowedGroups: groups,
	}
}

// Validate submits token to the Kubernetes TokenReview API, then authorizes
// the resulting identity via the allowlist or a SubjectAccessReview. Returns
// nil if the token is valid, authenticated, and authorized for admin access.
// Returns ErrUnauthenticated if the token is empty, ErrInvalidToken if the
// API rejects it, or ErrUnauthorized if neither authorization mechanism
// grants access.
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

	user := result.Status.User

	if v.isAllowlisted(user.Username, user.Groups) {
		return nil
	}

	allowed, err := v.checkSubjectAccessReview(ctx, user)
	if err != nil {
		return fmt.Errorf("%w: subject access review: %v", ErrUnauthorized, err)
	}
	if !allowed {
		return fmt.Errorf("%w: %s is not authorized to administer signet (grant via SIGNET_ADMIN_SUBJECTS or RBAC on %s/%s verb %s)",
			ErrUnauthorized, user.Username, adminSARGroup, adminSARResource, adminSARVerb)
	}
	return nil
}

func (v *TokenValidator) isAllowlisted(username string, groups []string) bool {
	if v.allowedUsers[username] {
		return true
	}
	for _, g := range groups {
		if v.allowedGroups[g] {
			return true
		}
	}
	return false
}

func (v *TokenValidator) checkSubjectAccessReview(ctx context.Context, user authv1.UserInfo) (bool, error) {
	extra := make(map[string]authzv1.ExtraValue, len(user.Extra))
	for k, vals := range user.Extra {
		extra[k] = authzv1.ExtraValue(vals)
	}

	sar := &authzv1.SubjectAccessReview{
		Spec: authzv1.SubjectAccessReviewSpec{
			User:   user.Username,
			Groups: user.Groups,
			UID:    user.UID,
			Extra:  extra,
			ResourceAttributes: &authzv1.ResourceAttributes{
				Group:    adminSARGroup,
				Resource: adminSARResource,
				Verb:     adminSARVerb,
			},
		},
	}

	result, err := v.client.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return false, err
	}
	return result.Status.Allowed, nil
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
