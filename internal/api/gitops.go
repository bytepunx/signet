package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/bytepunx/signet/internal/auth"
	"github.com/bytepunx/signet/internal/gitops"
	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GitOpsServer implements adminv1.GitOpsServiceServer.
type GitOpsServer struct {
	adminv1.UnimplementedGitOpsServiceServer
	store          gitopsStore
	keys           keyUnwrapper
	syncer         *gitops.Syncer
	webhookBaseURL string
	validator      tokenChecker
	environment    string // from SIGNET_ENVIRONMENT; empty = unscoped
}

// NewGitOpsServer constructs a GitOpsServer. environment scopes SOPS key
// operations to a specific deployment tier (e.g. "prod", "staging"); an empty
// string means no filtering is applied.
func NewGitOpsServer(
	st gitopsStore,
	keys keyUnwrapper,
	syncer *gitops.Syncer,
	webhookBaseURL string,
	validator tokenChecker,
	environment string,
) *GitOpsServer {
	return &GitOpsServer{
		store:          st,
		keys:           keys,
		syncer:         syncer,
		webhookBaseURL: webhookBaseURL,
		validator:      validator,
		environment:    environment,
	}
}

func (s *GitOpsServer) requireToken(ctx context.Context) error {
	token, err := auth.TokenFromMetadata(ctx)
	if err != nil {
		return toGRPCError(err)
	}
	if err := s.validator.Validate(ctx, token); err != nil {
		return toGRPCError(err)
	}
	return nil
}

// GetSOPSPublicKey returns the currently active age public key for this
// instance's environment.
func (s *GitOpsServer) GetSOPSPublicKey(ctx context.Context, _ *adminv1.GetSOPSPublicKeyRequest) (*adminv1.GetSOPSPublicKeyResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	key, err := s.store.GetActiveSOPSKey(ctx, s.environment)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &adminv1.GetSOPSPublicKeyResponse{
		PublicKey:   key.PublicKey,
		Fingerprint: ageFingerprint(key.PublicKey),
		CreatedAt:   key.CreatedAt.UTC().Format(time.RFC3339),
		Environment: key.Environment,
	}, nil
}

// RotateSOPSKey generates a new age keypair scoped to this instance's
// environment, deactivates the current environment-scoped key, and returns the
// new public key. The old key is retained for decryption until pruned.
func (s *GitOpsServer) RotateSOPSKey(ctx context.Context, _ *adminv1.RotateSOPSKeyRequest) (*adminv1.RotateSOPSKeyResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}

	// Only deactivate the active key for THIS environment.
	oldKey, _ := s.store.GetActiveSOPSKey(ctx, s.environment) // ignore not-found on first rotation
	if oldKey != nil {
		if err := s.store.DeactivateSOPSKey(ctx, oldKey.PublicKey); err != nil {
			return nil, toGRPCError(fmt.Errorf("deactivate old key: %w", err))
		}
	}

	pubKey, encPrivKey, err := gitops.GenerateAgeKey(s.keys)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate age key: %v", err)
	}

	newEntry := &store.SOPSKey{
		PublicKey:           pubKey,
		EncryptedPrivateKey: encPrivKey,
		Environment:         s.environment,
		IsActive:            true,
	}
	if err := s.store.PutSOPSKey(ctx, newEntry); err != nil {
		return nil, toGRPCError(err)
	}

	resp := &adminv1.RotateSOPSKeyResponse{
		NewPublicKey:    pubKey,
		NewFingerprint:  ageFingerprint(pubKey),
		NewEnvironment:  s.environment,
	}
	if oldKey != nil {
		resp.OldPublicKey = oldKey.PublicKey
	}
	return resp, nil
}

// ListSOPSKeys returns age keys visible to this instance's environment:
// keys tagged for this environment plus any global (unscoped) keys.
func (s *GitOpsServer) ListSOPSKeys(ctx context.Context, _ *adminv1.ListSOPSKeysRequest) (*adminv1.ListSOPSKeysResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	keys, err := s.store.ListSOPSKeys(ctx, s.environment)
	if err != nil {
		return nil, toGRPCError(err)
	}
	infos := make([]*adminv1.SOPSKeyInfo, len(keys))
	for i, k := range keys {
		info := &adminv1.SOPSKeyInfo{
			PublicKey:   k.PublicKey,
			Fingerprint: ageFingerprint(k.PublicKey),
			IsActive:    k.IsActive,
			CreatedAt:   k.CreatedAt.UTC().Format(time.RFC3339),
			Environment: k.Environment,
		}
		if k.DeactivatedAt != nil {
			info.DeactivatedAt = k.DeactivatedAt.UTC().Format(time.RFC3339)
		}
		infos[i] = info
	}
	return &adminv1.ListSOPSKeysResponse{Keys: infos}, nil
}

// PruneSOPSKey permanently deletes an inactive age key and its encrypted private key.
// Active keys cannot be pruned.
func (s *GitOpsServer) PruneSOPSKey(ctx context.Context, req *adminv1.PruneSOPSKeyRequest) (*adminv1.PruneSOPSKeyResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	key, err := s.store.GetActiveSOPSKey(ctx, s.environment)
	if err == nil && key != nil && key.PublicKey == req.GetPublicKey() {
		return nil, status.Error(codes.FailedPrecondition, "cannot prune the active key; rotate first")
	}
	if err := s.store.DeleteSOPSKey(ctx, req.GetPublicKey()); err != nil {
		return nil, toGRPCError(err)
	}
	return &adminv1.PruneSOPSKeyResponse{
		Message: fmt.Sprintf("key %s pruned", req.GetPublicKey()),
	}, nil
}

// RegisterRepository stores a new git repository configuration.
// Returns a webhook URL and a plaintext webhook secret (shown once only).
func (s *GitOpsServer) RegisterRepository(ctx context.Context, req *adminv1.RegisterRepositoryRequest) (*adminv1.RegisterRepositoryResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}

	// Generate a random webhook secret.
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, status.Errorf(codes.Internal, "generate webhook secret: %v", err)
	}
	webhookSecretHex := hex.EncodeToString(raw)

	// Encrypt the webhook secret and deploy key under the master key.
	var encWebhookSecret []byte
	if err := s.keys.Use(func(masterKey []byte) error {
		ct, err := icrypto.Encrypt(masterKey, []byte(webhookSecretHex))
		encWebhookSecret = ct
		return err
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "encrypt webhook secret: %v", err)
	}

	var encDeployKey []byte
	if err := s.keys.Use(func(masterKey []byte) error {
		ct, err := icrypto.Encrypt(masterKey, req.GetDeployKey())
		encDeployKey = ct
		return err
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "encrypt deploy key: %v", err)
	}

	branch := req.GetBranch()
	if branch == "" {
		branch = "main"
	}
	secretsPath := req.GetSecretsPath()
	if secretsPath == "" {
		secretsPath = "secrets/"
	}

	repo := &store.Repository{
		Name:                   req.GetName(),
		RepoURL:                req.GetRepoUrl(),
		Branch:                 branch,
		SecretsPath:            secretsPath,
		ConfigPath:             req.GetConfigPath(),
		EncryptedWebhookSecret: encWebhookSecret,
		EncryptedDeployKey:     encDeployKey,
	}
	if err := s.store.PutRepository(ctx, repo); err != nil {
		return nil, toGRPCError(err)
	}

	return &adminv1.RegisterRepositoryResponse{
		Id:            repo.ID,
		WebhookUrl:    s.webhookURL(repo.ID),
		WebhookSecret: webhookSecretHex,
	}, nil
}

// ListRepositories lists all registered git repositories.
func (s *GitOpsServer) ListRepositories(ctx context.Context, _ *adminv1.ListRepositoriesRequest) (*adminv1.ListRepositoriesResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	repos, err := s.store.ListRepositories(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	infos := make([]*adminv1.RepositoryInfo, len(repos))
	for i, r := range repos {
		info := &adminv1.RepositoryInfo{
			Id:          r.ID,
			Name:        r.Name,
			RepoUrl:     r.RepoURL,
			Branch:      r.Branch,
			SecretsPath: r.SecretsPath,
			ConfigPath:  r.ConfigPath,
			LastSyncSha: r.LastSyncSHA,
		}
		if r.LastSyncAt != nil {
			info.LastSyncAt = r.LastSyncAt.UTC().Format(time.RFC3339)
		}
		infos[i] = info
	}
	return &adminv1.ListRepositoriesResponse{Repositories: infos}, nil
}

// RemoveRepository deletes a repository registration (does not delete synced secrets).
func (s *GitOpsServer) RemoveRepository(ctx context.Context, req *adminv1.RemoveRepositoryRequest) (*adminv1.RemoveRepositoryResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	if err := s.store.DeleteRepository(ctx, req.GetId()); err != nil {
		return nil, toGRPCError(err)
	}
	return &adminv1.RemoveRepositoryResponse{
		Message: fmt.Sprintf("repository %s removed", req.GetId()),
	}, nil
}

// TriggerSync performs a full sync of the repository immediately.
func (s *GitOpsServer) TriggerSync(ctx context.Context, req *adminv1.TriggerSyncRequest) (*adminv1.TriggerSyncResponse, error) {
	if err := s.requireToken(ctx); err != nil {
		return nil, err
	}
	repo, err := s.store.GetRepository(ctx, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	result, err := s.syncer.FullSync(ctx, repo)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "sync failed: %v", err)
	}
	return &adminv1.TriggerSyncResponse{
		SecretsAdded:   int32(result.Added),
		SecretsUpdated: int32(result.Updated),
		SecretsDeleted: int32(result.Deleted),
		SyncSha:        result.SHA,
		ConfigsSynced:  int32(result.ConfigsSynced),
	}, nil
}

// SyncBundle receives a client-streamed tar.gz archive, extracts it to a temp
// directory, and runs the SOPS sync pass — identical to a FullSync but without
// requiring a remote git repository.
//
// Protocol: the first chunk must contain a SyncBundleHeader; subsequent chunks
// carry raw tar.gz bytes. The RPC is sealed (server remains operational); it
// can be called as many times as needed to refresh secrets from a local repo.
func (s *GitOpsServer) SyncBundle(stream adminv1.GitOpsService_SyncBundleServer) error {
	if err := s.requireToken(stream.Context()); err != nil {
		return err
	}

	// First chunk must be the header.
	first, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "receive header: %v", err)
	}
	hdr, ok := first.Payload.(*adminv1.SyncBundleChunk_Header)
	if !ok {
		return status.Error(codes.InvalidArgument, "first chunk must be SyncBundleHeader")
	}
	secretsPath := hdr.Header.GetSecretsPath()
	if secretsPath == "" {
		secretsPath = "secrets/"
	}
	headSHA := hdr.Header.GetHeadSha()
	configPath := hdr.Header.GetConfigPath()

	// Accumulate data chunks.
	var buf bytes.Buffer
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "receive chunk: %v", err)
		}
		data, ok := chunk.Payload.(*adminv1.SyncBundleChunk_Data)
		if !ok {
			return status.Error(codes.InvalidArgument, "expected data chunk after header")
		}
		buf.Write(data.Data)
	}

	// Extract to a temp directory.
	tmpDir, err := os.MkdirTemp("", "signet-bundle-*")
	if err != nil {
		return status.Errorf(codes.Internal, "create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractTarGz(&buf, tmpDir); err != nil {
		return status.Errorf(codes.InvalidArgument, "extract bundle: %v", err)
	}

	// Run the SOPS decrypt + store pass.
	result, err := s.syncer.SyncFromDir(stream.Context(), tmpDir, secretsPath, headSHA)
	if err != nil {
		return status.Errorf(codes.Internal, "sync: %v", err)
	}

	// Run the plain YAML config pass if a config_path was provided.
	configCount, configErr := s.syncer.SyncConfigFromDir(stream.Context(), tmpDir, configPath)
	if configErr != nil {
		slog.Warn("bundle config sync error", "err", configErr)
	}

	return stream.SendAndClose(&adminv1.SyncBundleResponse{
		SecretsAdded:   int32(result.Added),
		SecretsUpdated: int32(result.Updated),
		SecretsDeleted: int32(result.Deleted),
		SyncSha:        result.SHA,
		ConfigsSynced:  int32(configCount),
	})
}

// webhookURL builds the full webhook URL for a repository.
func (s *GitOpsServer) webhookURL(id string) string {
	base := strings.TrimSuffix(s.webhookBaseURL, "/")
	return base + "/webhook/github/" + id
}

// ageFingerprint returns an 8-byte hex fingerprint of an age public key.
func ageFingerprint(pubKey string) string {
	h := sha256.Sum256([]byte(pubKey))
	return hex.EncodeToString(h[:8])
}
