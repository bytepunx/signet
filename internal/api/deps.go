package api

import (
	"context"
	"encoding/json"

	"github.com/bytepunx/signet/internal/audit"
	"github.com/bytepunx/signet/internal/store"
	"github.com/bytepunx/signet/internal/unseal"
)

// secretFetcher is the subset of *store.Store used for reading secrets and config.
type secretFetcher interface {
	GetSecret(ctx context.Context, namespace, service, name string) (*store.Secret, error)
	GetServiceConfig(ctx context.Context, namespace, service string) (json.RawMessage, int, error)
	FetchServiceSecrets(ctx context.Context, namespace, service string) ([]store.Secret, error)
}

// keyUnwrapper is the subset of *crypto.KeyStore used for decrypting DEKs.
// Use calls fn with the master key bytes if the key is loaded; otherwise returns
// crypto.ErrKeyNotSet.
type keyUnwrapper interface {
	Use(fn func([]byte) error) error
}

// permissionChecker is the subset of *auth.Checker used for policy evaluation.
type permissionChecker interface {
	Allow(ctx context.Context, spiffeID, permission, namespace, service, secretName string) error
}

// auditRecorder is the subset of *audit.Writer used for recording access events.
type auditRecorder interface {
	Record(ctx context.Context, e audit.Entry) error
}

// unsealMgr is the subset of *unseal.Manager used by AdminServer.
type unsealMgr interface {
	UnsealWithKey(key []byte) error
	SubmitShare(share []byte) (unseal.Status, error)
	Seal()
	Status() unseal.Status
}

// tokenChecker is the subset of *auth.TokenValidator used for admin SA token validation.
type tokenChecker interface {
	Validate(ctx context.Context, token string) error
}

// sealChecker is satisfied by *unseal.Manager and lets handlers query seal state.
type sealChecker interface {
	Status() unseal.Status
}

// gitopsStore is the subset of *store.Store used by GitOpsServer.
type gitopsStore interface {
	PutSOPSKey(ctx context.Context, key *store.SOPSKey) error
	GetActiveSOPSKey(ctx context.Context, env string) (*store.SOPSKey, error)
	ListSOPSKeys(ctx context.Context, env string) ([]store.SOPSKey, error)
	DeactivateSOPSKey(ctx context.Context, pubKey string) error
	DeleteSOPSKey(ctx context.Context, pubKey string) error
	PutRepository(ctx context.Context, r *store.Repository) error
	GetRepository(ctx context.Context, id string) (*store.Repository, error)
	ListRepositories(ctx context.Context) ([]store.Repository, error)
	DeleteRepository(ctx context.Context, id string) error
}
