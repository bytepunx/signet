package gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	gogittransport "github.com/go-git/go-git/v5/plumbing/transport"
	gogitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"gopkg.in/yaml.v3"
)

// SyncResult summarises the outcome of a repository sync operation.
type SyncResult struct {
	Added         int
	Updated       int
	Deleted       int
	ConfigsSynced int
	SHA           string
}

// Syncer fetches a git repository, decrypts SOPS-encrypted secrets, and
// writes them to the secret store.
type Syncer struct {
	store       secretStore
	keys        keyUnwrapper
	bus         notifier
	environment string // SIGNET_ENVIRONMENT; empty = no filtering
}

// NewSyncer constructs a Syncer. bus may be nil if change notifications are
// not needed (e.g. during reconciliation before any watchers are active).
// environment scopes which SOPS age keys are loaded for decryption; an empty
// string means all keys are considered (backward-compatible default).
func NewSyncer(st secretStore, keys keyUnwrapper, bus notifier, environment string) *Syncer {
	return &Syncer{store: st, keys: keys, bus: bus, environment: environment}
}

// SyncFromPush handles an incremental sync driven by a GitHub push event.
// changedFiles and deletedFiles are repository-relative paths.
func (s *Syncer) SyncFromPush(ctx context.Context, repo *store.Repository, headSHA string, changedFiles, deletedFiles []string) (*SyncResult, error) {
	identities, err := s.loadIdentities(ctx)
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "signet-sync-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := s.cloneRepo(ctx, tmpDir, repo, headSHA); err != nil {
		return nil, err
	}

	result := &SyncResult{SHA: headSHA}

	for _, f := range changedFiles {
		// Try as SOPS-encrypted secret.
		ns, svc, name, err := ParseSecretPath(repo.SecretsPath, f)
		if err == nil {
			fullPath := filepath.Join(tmpDir, filepath.FromSlash(f))
			data, readErr := os.ReadFile(fullPath)
			if readErr != nil {
				slog.Error("read secret file", "path", f, "err", readErr)
				continue
			}
			if storeErr := s.storeSecret(ctx, ns, svc, name, data, identities); storeErr != nil {
				slog.Error("store secret", "path", f, "err", storeErr)
				continue
			}
			result.Added++
			if s.bus != nil {
				s.bus.Notify(ns, svc, name)
				s.bus.NotifyBundle(ns, svc)
			}
			continue
		}

		// Try as plain YAML config.
		if repo.ConfigPath != "" {
			ns, svc, configErr := ParseConfigPath(repo.ConfigPath, f)
			if configErr == nil {
				fullPath := filepath.Join(tmpDir, filepath.FromSlash(f))
				data, readErr := os.ReadFile(fullPath)
				if readErr != nil {
					slog.Error("read config file", "path", f, "err", readErr)
					continue
				}
				if storeErr := s.storeConfig(ctx, ns, svc, data); storeErr != nil {
					slog.Error("store config", "path", f, "err", storeErr)
					continue
				}
				result.ConfigsSynced++
				if s.bus != nil {
					s.bus.NotifyService(ns, svc)
					s.bus.NotifyBundle(ns, svc)
				}
				continue
			}
		}

		slog.Debug("skipping unrecognised file", "path", f)
	}

	for _, f := range deletedFiles {
		if ns, svc, name, err := ParseSecretPath(repo.SecretsPath, f); err == nil {
			if err := s.deleteSecret(ctx, ns, svc, name); err != nil {
				slog.Error("delete secret", "path", f, "err", err)
				continue
			}
			result.Deleted++
			if s.bus != nil {
				s.bus.Notify(ns, svc, name)
				s.bus.NotifyBundle(ns, svc)
			}
			continue
		}
		if repo.ConfigPath != "" {
			if ns, svc, err := ParseConfigPath(repo.ConfigPath, f); err == nil {
				type configDeleter interface {
					DeleteServiceConfig(ctx context.Context, namespace, service string) error
				}
				if d, ok := s.store.(configDeleter); ok {
					if err := d.DeleteServiceConfig(ctx, ns, svc); err != nil {
						slog.Error("delete config", "path", f, "err", err)
						continue
					}
					if s.bus != nil {
						s.bus.NotifyService(ns, svc)
						s.bus.NotifyBundle(ns, svc)
					}
				}
			}
		}
	}

	if err := s.store.UpdateSyncState(ctx, repo.ID, headSHA, time.Now().UTC()); err != nil {
		slog.Error("update sync state", "repo", repo.Name, "err", err)
	}
	return result, nil
}

// FullSync clones the repository and syncs every YAML file under SecretsPath.
// Used for initial sync and reconciliation.
func (s *Syncer) FullSync(ctx context.Context, repo *store.Repository) (*SyncResult, error) {
	tmpDir, err := os.MkdirTemp("", "signet-sync-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	headSHA, err := s.cloneRepoAtHead(ctx, tmpDir, repo)
	if err != nil {
		return nil, err
	}

	result, err := s.SyncFromDir(ctx, tmpDir, repo.SecretsPath, headSHA)
	if err != nil {
		return nil, err
	}

	configCount, configErr := s.SyncConfigFromDir(ctx, tmpDir, repo.ConfigPath)
	if configErr != nil {
		slog.Warn("config sync error", "repo", repo.Name, "err", configErr)
	}
	result.ConfigsSynced = configCount

	if err := s.store.UpdateSyncState(ctx, repo.ID, headSHA, time.Now().UTC()); err != nil {
		slog.Error("update sync state", "repo", repo.Name, "err", err)
	}
	return result, nil
}

// SyncFromDir processes every SOPS-encrypted YAML file under secretsPath
// within an already-populated directory. headSHA is recorded on the result
// for audit purposes; it may be empty for non-git sources.
//
// This is the core of both FullSync (which clones first) and SyncBundle
// (which extracts a tar archive first).
func (s *Syncer) SyncFromDir(ctx context.Context, dir, secretsPath, headSHA string) (*SyncResult, error) {
	identities, err := s.loadIdentities(ctx)
	if err != nil {
		return nil, err
	}

	result := &SyncResult{SHA: headSHA}
	secretsDir := filepath.Join(dir, filepath.FromSlash(secretsPath))

	err = filepath.WalkDir(secretsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return walkErr
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)

		ns, svc, name, err := ParseSecretPath(secretsPath, rel)
		if err != nil {
			return nil //nolint:nilerr // not a secret file; continue walking rather than abort
		}
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Error("read secret file", "path", rel, "err", err)
			return nil
		}
		if err := s.storeSecret(ctx, ns, svc, name, data, identities); err != nil {
			slog.Error("store secret", "path", rel, "err", err)
			return nil
		}
		result.Added++
		if s.bus != nil {
			s.bus.Notify(ns, svc, name)
			s.bus.NotifyBundle(ns, svc)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk secrets dir: %w", err)
	}
	return result, nil
}

// loadIdentities fetches the age keys applicable to this instance's environment
// and decrypts them. When environment is empty all keys are loaded.
func (s *Syncer) loadIdentities(ctx context.Context) ([]age.Identity, error) {
	keys, err := s.store.ListSOPSKeys(ctx, s.environment)
	if err != nil {
		return nil, fmt.Errorf("list sops keys: %w", err)
	}
	if len(keys) == 0 {
		if s.environment != "" {
			return nil, fmt.Errorf("no age keys configured for environment %q; run 'signet sops-key rotate' to generate one", s.environment)
		}
		return nil, fmt.Errorf("no age keys configured; run 'signet sops-key rotate' to generate one")
	}
	ids := make([]age.Identity, 0, len(keys))
	for _, k := range keys {
		id, err := DecryptAgeKey(s.keys, k.EncryptedPrivateKey)
		if err != nil {
			slog.Warn("skip unusable age key", "pubkey", k.PublicKey, "err", err)
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("could not decrypt any age keys")
	}
	return ids, nil
}

// storeSecret decrypts a SOPS file and writes the secret to the store.
func (s *Syncer) storeSecret(ctx context.Context, namespace, service, name string, data []byte, identities []age.Identity) error {
	plaintext, err := DecryptFile(data, identities)
	if err != nil {
		return fmt.Errorf("sops decrypt: %w", err)
	}

	// Encrypt the plaintext under a fresh per-secret DEK wrapped by the master key.
	dek, err := icrypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("generate dek: %w", err)
	}
	defer ZeroBytes(dek)

	ciphertext, err := icrypto.Encrypt(dek, plaintext)
	ZeroBytes(plaintext)
	if err != nil {
		return fmt.Errorf("encrypt secret: %w", err)
	}

	var encDEK []byte
	if err := s.keys.Use(func(masterKey []byte) error {
		var werr error
		encDEK, werr = icrypto.WrapKey(masterKey, dek)
		return werr
	}); err != nil {
		return fmt.Errorf("wrap dek: %w", err)
	}

	return s.store.PutSecret(ctx, &store.Secret{
		Namespace:    namespace,
		Service:      service,
		Name:         name,
		EncryptedDEK: encDEK,
		Ciphertext:   ciphertext,
	})
}

// deleteSecret removes a secret from the store, tolerating not-found.
func (s *Syncer) deleteSecret(ctx context.Context, namespace, service, name string) error {
	// Use a concrete store.Store to call DeleteSecret; the interface is minimal.
	// We rely on the runtime assertion in loadIdentities for type safety.
	type deleter interface {
		DeleteSecret(ctx context.Context, namespace, service, name string) error
	}
	if d, ok := s.store.(deleter); ok {
		return d.DeleteSecret(ctx, namespace, service, name)
	}
	return nil
}

// cloneRepo clones at the specific headSHA into dir using the repo's deploy key.
func (s *Syncer) cloneRepo(ctx context.Context, dir string, repo *store.Repository, headSHA string) error {
	auth, err := s.deployKeyAuth(repo)
	if err != nil {
		return err
	}
	r, err := gogit.PlainCloneContext(ctx, dir, false, &gogit.CloneOptions{
		URL:           repo.RepoURL,
		ReferenceName: plumbing.NewBranchReferenceName(repo.Branch),
		SingleBranch:  true,
		Depth:         1,
		Auth:          auth,
	})
	if err != nil {
		return fmt.Errorf("clone repo %s: %w", repo.RepoURL, err)
	}
	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("open worktree for %s: %w", repo.RepoURL, err)
	}
	if err := w.Checkout(&gogit.CheckoutOptions{Hash: plumbing.NewHash(headSHA)}); err != nil {
		return fmt.Errorf("checkout %s at %s: %w", repo.RepoURL, headSHA, err)
	}
	return nil
}

// cloneRepoAtHead clones the default branch and returns the HEAD commit SHA.
func (s *Syncer) cloneRepoAtHead(ctx context.Context, dir string, repo *store.Repository) (string, error) {
	auth, err := s.deployKeyAuth(repo)
	if err != nil {
		return "", err
	}
	r, err := gogit.PlainCloneContext(ctx, dir, false, &gogit.CloneOptions{
		URL:           repo.RepoURL,
		ReferenceName: plumbing.NewBranchReferenceName(repo.Branch),
		SingleBranch:  true,
		Depth:         1,
		Auth:          auth,
	})
	if err != nil {
		return "", fmt.Errorf("clone repo %s: %w", repo.RepoURL, err)
	}
	ref, err := r.Head()
	if err != nil {
		return "", fmt.Errorf("resolve HEAD: %w", err)
	}
	return ref.Hash().String(), nil
}

// deployKeyAuth decrypts the repo's SSH deploy key and returns go-git auth.
func (s *Syncer) deployKeyAuth(repo *store.Repository) (gogittransport.AuthMethod, error) {
	var pemBytes []byte
	if err := s.keys.Use(func(masterKey []byte) error {
		plain, err := icrypto.Decrypt(masterKey, repo.EncryptedDeployKey)
		if err != nil {
			return err
		}
		pemBytes = plain
		return nil
	}); err != nil {
		return nil, fmt.Errorf("decrypt deploy key: %w", err)
	}
	defer ZeroBytes(pemBytes)

	auth, err := gogitssh.NewPublicKeys("git", pemBytes, "")
	if err != nil {
		return nil, fmt.Errorf("parse deploy key: %w", err)
	}
	return auth, nil
}

// SyncConfigFromDir processes every plain YAML file under configPath within an
// already-populated directory. Files at <configPath>/<namespace>/<service>.yaml
// are parsed as nested YAML maps, converted to JSON, and stored as service configs.
// Returns the number of configs synced and any walk error.
func (s *Syncer) SyncConfigFromDir(ctx context.Context, dir, configPath string) (int, error) {
	if configPath == "" {
		return 0, nil
	}
	configDir := filepath.Join(dir, filepath.FromSlash(configPath))
	if _, statErr := os.Stat(configDir); os.IsNotExist(statErr) {
		slog.Debug("config directory not present, skipping config sync", "path", configPath)
		return 0, nil
	}

	var count int
	err := filepath.WalkDir(configDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return walkErr
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)

		ns, svc, err := ParseConfigPath(configPath, rel)
		if err != nil {
			return nil //nolint:nilerr // not a config file; continue walking rather than abort
		}
		data, err := os.ReadFile(path)
		if err != nil {
			slog.Error("read config file", "path", rel, "err", err)
			return nil
		}
		if err := s.storeConfig(ctx, ns, svc, data); err != nil {
			slog.Error("store config", "path", rel, "err", err)
			return nil
		}
		count++
		if s.bus != nil {
			s.bus.NotifyService(ns, svc)
			s.bus.NotifyBundle(ns, svc)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walk config dir: %w", err)
	}
	return count, nil
}

// storeConfig parses a plain YAML config file and stores it as a JSON document.
func (s *Syncer) storeConfig(ctx context.Context, namespace, service string, data []byte) error {
	var raw interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}
	normalized := normalizeForJSON(raw)
	m, ok := normalized.(map[string]interface{})
	if !ok {
		return fmt.Errorf("config file must be a YAML mapping, got %T", raw)
	}
	content, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	return s.store.PutServiceConfig(ctx, namespace, service, content)
}

// normalizeForJSON converts yaml.v3-produced values to JSON-safe equivalents.
// yaml.v3 can produce time.Time for unquoted date literals; these are converted
// to RFC3339 strings. All other JSON-compatible types pass through unchanged.
func normalizeForJSON(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, v := range val {
			out[k] = normalizeForJSON(v)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, v := range val {
			out[i] = normalizeForJSON(v)
		}
		return out
	case time.Time:
		return val.UTC().Format(time.RFC3339)
	default:
		return val
	}
}

// ZeroBytes overwrites b with zeros to clear sensitive data from memory.
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
