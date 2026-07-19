package gitops

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bytepunx/signet/internal/store"
)

// secretStore is satisfied by *store.Store for the sync path.
type secretStore interface {
	PutSecret(ctx context.Context, s *store.Secret) error
	GetSecret(ctx context.Context, namespace, service, name string) (*store.Secret, error)
	DeleteSecret(ctx context.Context, namespace, service, name string) error
	ListSOPSKeys(ctx context.Context, env string) ([]store.SOPSKey, error)
	GetRepository(ctx context.Context, id string) (*store.Repository, error)
	ListRepositories(ctx context.Context) ([]store.Repository, error)
	UpdateSyncState(ctx context.Context, id, sha string, at time.Time) error
	PutServiceConfig(ctx context.Context, namespace, service string, content json.RawMessage, repoID string) error
	DeleteServiceConfig(ctx context.Context, namespace, service string) error
	GetActiveKEK(ctx context.Context) (*store.KEK, error)
	PutKEK(ctx context.Context, k *store.KEK) error
	ListSecretKeysForRepo(ctx context.Context, repoID string) ([]store.SecretKey, error)
	ListConfigKeysForRepo(ctx context.Context, repoID string) ([]store.ConfigKey, error)
	UpdateSecretRepoID(ctx context.Context, namespace, service, name, repoID string) error
}

// keyUnwrapper is satisfied by *crypto.KeyStore.
type keyUnwrapper interface {
	Use(fn func([]byte) error) error
}

// notifier is satisfied by *api.Bus — minimal interface to avoid import cycle.
type notifier interface {
	Notify(namespace, service, name string)
	NotifyService(namespace, service string)
	NotifyBundle(namespace, service string)
}
