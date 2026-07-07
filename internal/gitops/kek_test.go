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

// statefulKEKStore is a secretStore fake that actually persists KEKs and
// secrets in memory, so activeKEK's bootstrap-then-reuse behavior and the
// M-4 dedup logic in storeSecret/isUnchanged can be exercised.
type statefulKEKStore struct {
	active     *store.KEK
	puts       int
	secrets    map[string]*store.Secret
	putSecrets int
}

func secretKey(namespace, service, name string) string {
	return namespace + "/" + service + "/" + name
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
	return nil, nil
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
func (s *statefulKEKStore) DeleteSecret(_ context.Context, _, _, _ string) error { return nil }
func (s *statefulKEKStore) GetRepository(_ context.Context, _ string) (*store.Repository, error) {
	return nil, nil
}
func (s *statefulKEKStore) ListRepositories(_ context.Context) ([]store.Repository, error) {
	return nil, nil
}
func (s *statefulKEKStore) UpdateSyncState(_ context.Context, _, _ string, _ time.Time) error {
	return nil
}
func (s *statefulKEKStore) PutServiceConfig(_ context.Context, _, _ string, _ json.RawMessage) error {
	return nil
}
func (s *statefulKEKStore) DeleteServiceConfig(_ context.Context, _, _ string) error { return nil }

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
