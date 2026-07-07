package auth

import (
	"context"
	"testing"

	authv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

// tokenReviewReactor makes the fake clientset respond to TokenReview creation
// with a fixed authenticated/username/groups result.
func tokenReviewReactor(authenticated bool, username string, groups []string) k8stesting.ReactionFunc {
	return func(action k8stesting.Action) (bool, runtime.Object, error) {
		review := action.(k8stesting.CreateAction).GetObject().(*authv1.TokenReview)
		review.Status = authv1.TokenReviewStatus{
			Authenticated: authenticated,
			User:          authv1.UserInfo{Username: username, Groups: groups},
		}
		return true, review, nil
	}
}

// subjectAccessReviewReactor makes the fake clientset respond to
// SubjectAccessReview creation with a fixed allowed result.
func subjectAccessReviewReactor(allowed bool) k8stesting.ReactionFunc {
	return func(action k8stesting.Action) (bool, runtime.Object, error) {
		sar := action.(k8stesting.CreateAction).GetObject().(*authzv1.SubjectAccessReview)
		sar.Status = authzv1.SubjectAccessReviewStatus{Allowed: allowed}
		return true, sar, nil
	}
}

func TestTokenValidator_EmptyToken(t *testing.T) {
	v := NewTokenValidator(fake.NewClientset(), nil, nil)
	err := v.Validate(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthenticated)
}

func TestTokenValidator_NotAuthenticated(t *testing.T) {
	client := fake.NewClientset()
	client.Fake.PrependReactor("create", "tokenreviews", tokenReviewReactor(false, "", nil))
	v := NewTokenValidator(client, nil, nil)

	err := v.Validate(context.Background(), "bad-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokenValidator_AuthenticatedButNotAuthorized(t *testing.T) {
	client := fake.NewClientset()
	client.Fake.PrependReactor("create", "tokenreviews",
		tokenReviewReactor(true, "system:serviceaccount:default:some-pod", nil))
	client.Fake.PrependReactor("create", "subjectaccessreviews", subjectAccessReviewReactor(false))
	v := NewTokenValidator(client, nil, nil)

	err := v.Validate(context.Background(), "valid-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestTokenValidator_AuthorizedViaAllowlist_Username(t *testing.T) {
	client := fake.NewClientset()
	client.Fake.PrependReactor("create", "tokenreviews",
		tokenReviewReactor(true, "system:serviceaccount:signet:signet-admin", nil))
	// No SAR reactor registered — if the allowlist fast-path is not taken,
	// this call would fail with "no reactor found" and the test would error.
	v := NewTokenValidator(client, nil, []string{"serviceaccount:signet:signet-admin"})

	err := v.Validate(context.Background(), "valid-token")
	require.NoError(t, err)
}

func TestTokenValidator_AuthorizedViaAllowlist_Group(t *testing.T) {
	client := fake.NewClientset()
	client.Fake.PrependReactor("create", "tokenreviews",
		tokenReviewReactor(true, "system:serviceaccount:signet:some-sa", []string{"signet-operators"}))
	v := NewTokenValidator(client, nil, []string{"group:signet-operators"})

	err := v.Validate(context.Background(), "valid-token")
	require.NoError(t, err)
}

func TestTokenValidator_NotOnAllowlist_FallsBackToSAR_Allowed(t *testing.T) {
	client := fake.NewClientset()
	client.Fake.PrependReactor("create", "tokenreviews",
		tokenReviewReactor(true, "system:serviceaccount:signet:other-sa", nil))
	client.Fake.PrependReactor("create", "subjectaccessreviews", subjectAccessReviewReactor(true))
	v := NewTokenValidator(client, nil, []string{"serviceaccount:signet:signet-admin"})

	err := v.Validate(context.Background(), "valid-token")
	require.NoError(t, err, "identity not on the allowlist must still be authorized via RBAC/SAR")
}

func TestTokenValidator_SubjectAccessReviewAPIError(t *testing.T) {
	client := fake.NewClientset()
	client.Fake.PrependReactor("create", "tokenreviews",
		tokenReviewReactor(true, "system:serviceaccount:default:some-pod", nil))
	client.Fake.PrependReactor("create", "subjectaccessreviews",
		func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, assert.AnError
		})
	v := NewTokenValidator(client, nil, nil)

	err := v.Validate(context.Background(), "valid-token")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestNewTokenValidator_ParsesAllowedSubjects(t *testing.T) {
	v := NewTokenValidator(fake.NewClientset(), nil, []string{
		"serviceaccount:signet:signet-admin",
		"group:signet-operators",
		"malformed-entry-ignored",
		"serviceaccount:missing-name:",
	})
	assert.True(t, v.allowedUsers["system:serviceaccount:signet:signet-admin"])
	assert.True(t, v.allowedGroups["signet-operators"])
	assert.Len(t, v.allowedUsers, 1)
	assert.Len(t, v.allowedGroups, 1)
}
