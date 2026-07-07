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
	GetKEKByID(ctx context.Context, id string) (*store.KEK, error)
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
	// RotateMasterKey swaps the in-memory master key for a new one while the
	// server remains Unsealed throughout. Callers must have already re-wrapped
	// every KEK (and the key-check value) under newKey before calling this.
	RotateMasterKey(newKey []byte) error
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

// adminStore is the subset of *store.Store used by AdminServer for the
// key-check value, KEK lifecycle, and master key rotation.
type adminStore interface {
	GetKeyCheckValue(ctx context.Context) ([]byte, error)
	PutKeyCheckValue(ctx context.Context, ciphertext []byte) error
	GetActiveKEK(ctx context.Context) (*store.KEK, error)
	GetKEKByID(ctx context.Context, id string) (*store.KEK, error)
	PutKEK(ctx context.Context, k *store.KEK) error
	ListKEKs(ctx context.Context) ([]store.KEK, error)
	DeactivateKEK(ctx context.Context, id string) error
	DeleteKEK(ctx context.Context, id string) error
	CountSecretsUsingKEK(ctx context.Context, id string) (int, error)
	ListSecretKeyRefs(ctx context.Context) ([]store.SecretKeyRef, error)
	UpdateSecretDEK(ctx context.Context, namespace, service, name string, version int, newEncDEK []byte, newKEKID string) error
	RewrapKEKsAndKCV(ctx context.Context, kekUpdates []store.KEKRewrap, newKCV []byte) error
}
