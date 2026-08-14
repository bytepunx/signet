package gitops

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// statefulKEKStore is a secretStore fake that actually persists KEKs,
// secrets, and configs in memory (including RepoID attribution and real
// deletion), so activeKEK's bootstrap-then-reuse behavior and the M-4 dedup
// logic in storeSecret/isUnchanged can be exercised without a real database.
type statefulKEKStore struct {
	active        *store.KEK
	puts          int
	secrets       map[string]*store.Secret
	putSecrets    int
	configs       map[string]json.RawMessage
	configSynced  map[string]json.RawMessage
	configRepoID  map[string]string
	configVersion map[string]int
	sopsKeys      []store.SOPSKey
}

func secretKey(namespace, service, name string) string {
	return namespace + "/" + service + "/" + name
}

func configKey(namespace, service string) string {
	return namespace + "/" + service
}

func (s *statefulKEKStore) GetActiveKEK(_ context.Context) (*store.KEK, error) {
	if s.active == nil {
		return nil, store.ErrNotFound
	}
	return s.active, nil
}
func (s *statefulKEKStore) PutKEK(_ context.Context, k *store.KEK) error {
	k.ID = "kek-1"
	s.puts++
	cp := *k
	s.active = &cp
	return nil
}
func (s *statefulKEKStore) GetSecret(_ context.Context, namespace, service, name string) (*store.Secret, error) {
	sec, ok := s.secrets[secretKey(namespace, service, name)]
	if !ok {
		return nil, store.ErrNotFound
	}
	return sec, nil
}
func (s *statefulKEKStore) ListSOPSKeys(_ context.Context, _ string) ([]store.SOPSKey, error) {
	return s.sopsKeys, nil
}
func (s *statefulKEKStore) PutSecret(_ context.Context, sec *store.Secret) error {
	s.putSecrets++
	if s.secrets == nil {
		s.secrets = make(map[string]*store.Secret)
	}
	cp := *sec
	s.secrets[secretKey(sec.Namespace, sec.Service, sec.Name)] = &cp
	return nil
}
func (s *statefulKEKStore) DeleteSecret(_ context.Context, namespace, service, name string) error {
	delete(s.secrets, secretKey(namespace, service, name))
	return nil
}
func (s *statefulKEKStore) GetRepository(_ context.Context, _ string) (*store.Repository, error) {
	return nil, nil
}
func (s *statefulKEKStore) ListRepositories(_ context.Context) ([]store.Repository, error) {
	return nil, nil
}
func (s *statefulKEKStore) UpdateSyncState(_ context.Context, _, _ string, _ time.Time) error {
	return nil
}

// SyncServiceConfig mirrors store.Store.SyncServiceConfig's contract (see
// its doc comment) closely enough to exercise mergeConfigSync end to end:
// insert directly if the config doesn't exist yet, otherwise call merge
// with (synced, live, git) and apply its decision.
func (s *statefulKEKStore) SyncServiceConfig(
	_ context.Context, namespace, service string, gitContent json.RawMessage, repoID string,
	merge func(syncedContent, liveContent, gitContent json.RawMessage) (newContent json.RawMessage, conflict bool, err error),
) (version int, conflict bool, err error) {
	if s.configs == nil {
		s.configs = make(map[string]json.RawMessage)
		s.configSynced = make(map[string]json.RawMessage)
		s.configRepoID = make(map[string]string)
		s.configVersion = make(map[string]int)
	}
	k := configKey(namespace, service)
	live, exists := s.configs[k]
	if !exists {
		s.configs[k] = gitContent
		s.configSynced[k] = gitContent
		s.configRepoID[k] = repoID
		s.configVersion[k] = 1
		return 1, false, nil
	}

	newContent, conflict, err := merge(s.configSynced[k], live, gitContent)
	if err != nil {
		return 0, false, err
	}
	if newContent == nil {
		return 0, conflict, nil
	}
	s.configs[k] = newContent
	s.configSynced[k] = gitContent
	s.configRepoID[k] = repoID
	s.configVersion[k]++
	return s.configVersion[k], false, nil
}
func (s *statefulKEKStore) DeleteServiceConfig(_ context.Context, namespace, service string) error {
	k := configKey(namespace, service)
	delete(s.configs, k)
	delete(s.configSynced, k)
	delete(s.configRepoID, k)
	delete(s.configVersion, k)
	return nil
}
func (s *statefulKEKStore) UpdateSecretRepoID(_ context.Context, namespace, service, name, repoID string) error {
	sec, ok := s.secrets[secretKey(namespace, service, name)]
	if !ok {
		return store.ErrNotFound
	}
	sec.RepoID = repoID
	return nil
}

func TestActiveKEK_BootstrapsWhenNoneExists(t *testing.T) {
	st := &statefulKEKStore{}
	keys := &mockKeys{}

	id, kek, err := activeKEK(context.Background(), st, keys)
	require.NoError(t, err)
	assert.Equal(t, "kek-1", id)
	assert.Len(t, kek, icrypto.KeySize)
	assert.Equal(t, 1, st.puts, "bootstrap must persist exactly one KEK")
}

func TestActiveKEK_ReusesExistingActiveKEK(t *testing.T) {
	st := &statefulKEKStore{}
	keys := &mockKeys{}

	id1, kek1, err := activeKEK(context.Background(), st, keys)
	require.NoError(t, err)

	id2, kek2, err := activeKEK(context.Background(), st, keys)
	require.NoError(t, err)

	assert.Equal(t, id1, id2)
	assert.Equal(t, kek1, kek2)
	assert.Equal(t, 1, st.puts, "a second call must not bootstrap a new KEK")
}

func TestActiveKEK_UnwrapFailurePropagates(t *testing.T) {
	st := &statefulKEKStore{active: &store.KEK{ID: "kek-1", WrappedKEK: []byte("not-valid-ciphertext")}}
	keys := &mockKeys{}

	_, _, err := activeKEK(context.Background(), st, keys)
	require.Error(t, err)
}

func TestActiveKEK_WrongMasterKeyCannotUnwrapBootstrappedKEK(t *testing.T) {
	st := &statefulKEKStore{}

	// Bootstrap under one master key.
	_, _, err := activeKEK(context.Background(), st, &mockKeys{})
	require.NoError(t, err)

	// A different master key must not be able to unwrap it.
	wrongKey, err := icrypto.GenerateKey()
	require.NoError(t, err)
	_, _, err = activeKEK(context.Background(), st, &fixedKeyUnwrapper{key: wrongKey})
	require.Error(t, err)
}
