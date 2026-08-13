package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/bytepunx/signet/internal/audit"
	"github.com/bytepunx/signet/internal/auth"
	icrypto "github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/gitops"
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
	checker        permissionChecker
	// audit records PatchServiceConfig writes directly (bytepunx/signet#38) —
	// unlike SyncBundle's secret/config writes, a patch never goes through
	// gitops.Syncer (there's no sync pass, no repo, no bundle to walk), so it
	// can't rely on Syncer's own audit wiring (bytepunx/signet#25) and needs
	// its own.
	audit auditRecorder
	// bus notifies WatchServiceConfig/WatchServiceBundle subscribers after a
	// successful patch, the same way gitops.Syncer's storeConfig does for a
	// git/bundle-driven config write — a caller watching a service's config
	// should see a PatchServiceConfig change just as promptly.
	bus         *Bus
	environment string // from SIGNET_ENVIRONMENT; empty = unscoped
}

// NewGitOpsServer constructs a GitOpsServer. environment scopes SOPS key
// operations to a specific deployment tier (e.g. "prod", "staging"); an empty
// string means no filtering is applied. checker authorizes SyncBundle/
// PatchServiceConfig calls made over a workload's own SPIFFE mTLS identity
// rather than an admin bearer token — see authorizeGitOpsWrite.
func NewGitOpsServer(
	st gitopsStore,
	keys keyUnwrapper,
	syncer *gitops.Syncer,
	webhookBaseURL string,
	validator tokenChecker,
	checker permissionChecker,
	audit auditRecorder,
	bus *Bus,
	environment string,
) *GitOpsServer {
	return &GitOpsServer{
		store:          st,
		keys:           keys,
		syncer:         syncer,
		webhookBaseURL: webhookBaseURL,
		validator:      validator,
		checker:        checker,
		audit:          audit,
		bus:            bus,
		environment:    environment,
	}
}

func (s *GitOpsServer) requireToken(ctx context.Context) error {
	_, err := s.requireTokenIdentity(ctx)
	return err
}

// requireTokenIdentity is requireToken's identity-returning counterpart, for
// callers that need to attribute a write to the acting admin identity (e.g.
// TriggerSync — see bytepunx/signet#25's audit trail for GitOps writes).
func (s *GitOpsServer) requireTokenIdentity(ctx context.Context) (identity string, err error) {
	token, err := auth.TokenFromMetadata(ctx)
	if err != nil {
		return "", toGRPCError(err)
	}
	identity, err = s.validator.Validate(ctx, token)
	if err != nil {
		return "", toGRPCError(err)
	}
	return identity, nil
}

// authorizeGitOpsWrite authenticates a workload-reachable GitOpsService write
// (SyncBundle, PatchServiceConfig — bytepunx/signet#23, #38) via whichever
// credential the caller actually presented: an admin bearer token (checked
// exactly like requireToken, granting the same unscoped access these RPCs
// have always had via the admin listener), or — when no token is present at
// all — the caller's own verified SPIFFE mTLS identity. A malformed or
// invalid token fails immediately rather than falling back to the SPIFFE
// path, so a caller that clearly attempted admin auth gets a clear error
// instead of a confusing scope-mismatch one. actor is always non-empty on
// success — the admin's Kubernetes identity or the caller's SPIFFE ID — for
// attributing the resulting write in the audit log (bytepunx/signet#25).
// scoped is true only for the SPIFFE path, meaning the caller must also pass
// an explicit per-call authorization check (e.g. validateBundleScope, or a
// checker.Allow "put" check) before anything is written. rpcName appears in
// the error message when neither credential is present.
func (s *GitOpsServer) authorizeGitOpsWrite(ctx context.Context, rpcName string) (actor string, scoped bool, err error) {
	if token, tokenErr := auth.TokenFromMetadata(ctx); tokenErr == nil {
		identity, err := s.validator.Validate(ctx, token)
		if err != nil {
			return "", false, toGRPCError(err)
		}
		return identity, false, nil
	}

	id, idErr := auth.SPIFFEIDFromContext(ctx)
	if idErr != nil {
		return "", false, toGRPCError(fmt.Errorf(
			"%w: %s requires either an admin bearer token or a verified workload mTLS identity",
			auth.ErrUnauthenticated, rpcName))
	}
	return id, true, nil
}

// validateBundleScope enforces the SPIFFE-authenticated SyncBundle path's
// authorization boundary: every secret/config file under secretsPath/
// configPath that resolves to a valid namespace/service (via
// gitops.ParseSecretPath/ParseConfigPath — files that don't parse are left
// for the syncer's own skip handling, see bytepunx/signet#22) must pass
// checker.Allow for the "put" permission, exactly like a read would need
// "get". In practice this means the caller's own namespace/service via the
// checker's exact-match convention, or an explicit policy grant for
// anything broader. The first file that fails aborts the whole call before
// SyncFromDir/SyncConfigFromDir write anything — a partially-authorized
// bundle must not partially apply.
func (s *GitOpsServer) validateBundleScope(ctx context.Context, dir, secretsPath, configPath, spiffeID string) error {
	walk := func(root string, parse func(root, rel string) (namespace, service, name string, err error)) error {
		if root == "" {
			return nil
		}
		base := filepath.Join(dir, filepath.FromSlash(root))
		return filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".yaml") {
				return walkErr
			}
			rel, _ := filepath.Rel(dir, path)
			rel = filepath.ToSlash(rel)

			ns, svc, name, parseErr := parse(root, rel)
			if parseErr != nil {
				return nil //nolint:nilerr // not our concern here; the syncer surfaces this separately
			}
			if allowErr := s.checker.Allow(ctx, spiffeID, "put", ns, svc, name); allowErr != nil {
				return toGRPCError(fmt.Errorf("%w: %s may not push %s", auth.ErrUnauthorized, spiffeID, rel))
			}
			return nil
		})
	}

	if err := walk(secretsPath, gitops.ParseSecretPath); err != nil {
		return err
	}
	return walk(configPath, func(root, rel string) (namespace, service, name string, err error) {
		namespace, service, err = gitops.ParseConfigPath(root, rel)
		return namespace, service, "", err
	})
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
		NewPublicKey:   pubKey,
		NewFingerprint: ageFingerprint(pubKey),
		NewEnvironment: s.environment,
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

	// Encrypt the webhook secret and deploy key under the master key, bound
	// via AAD to this repository's name so the ciphertext cannot be swapped
	// with another repository's by a party with database write access.
	var encWebhookSecret []byte
	if err := s.keys.Use(func(masterKey []byte) error {
		ct, err := icrypto.Encrypt(masterKey, []byte(webhookSecretHex), icrypto.BindAAD(icrypto.AADRepoWebhookSecret, req.GetName()))
		encWebhookSecret = ct
		return err
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "encrypt webhook secret: %v", err)
	}

	var encDeployKey []byte
	if err := s.keys.Use(func(masterKey []byte) error {
		ct, err := icrypto.Encrypt(masterKey, req.GetDeployKey(), icrypto.BindAAD(icrypto.AADRepoDeployKey, req.GetName()))
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
		}
		if r.LastSyncSHA != nil {
			info.LastSyncSha = *r.LastSyncSHA
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
	actor, err := s.requireTokenIdentity(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := s.store.GetRepository(ctx, req.GetId())
	if err != nil {
		return nil, toGRPCError(err)
	}
	result, err := s.syncer.FullSync(ctx, repo, actor)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "sync failed: %v", err)
	}
	return &adminv1.TriggerSyncResponse{
		SecretsAdded:   int32(result.Added),
		SecretsUpdated: int32(result.Updated),
		SecretsDeleted: int32(result.Deleted),
		SyncSha:        result.SHA,
		ConfigsSynced:  int32(result.ConfigsSynced),
		Errors:         skippedToErrors(result.Skipped),
	}, nil
}

// skippedToErrors formats repo-relative paths skipped for not matching the
// path-depth convention (SyncResult.Skipped) as messages for the response's
// Errors field, so "signet repo sync" and equivalent RPC callers see them
// instead of a sync that silently reports success (see bytepunx/signet#22).
func skippedToErrors(skipped []string) []string {
	if len(skipped) == 0 {
		return nil
	}
	errs := make([]string, len(skipped))
	for i, path := range skipped {
		errs[i] = fmt.Sprintf("skipped %s: does not match the required <namespace>/<service>[/<name>].yaml path depth", path)
	}
	return errs
}

// SyncBundle receives a client-streamed tar.gz archive, extracts it to a temp
// directory, and runs the SOPS sync pass — identical to a FullSync but without
// requiring a remote git repository.
//
// Protocol: the first chunk must contain a SyncBundleHeader; subsequent chunks
// carry raw tar.gz bytes. The RPC is sealed (server remains operational); it
// can be called as many times as needed to refresh secrets from a local repo.
//
// SyncBundle is reachable two ways (see internal/server): via the admin
// listener with a bearer token (full access, as originally designed), and
// via the workload mTLS listener with no token at all, authenticated by the
// caller's own SPIFFE identity instead (bytepunx/signet#23) — e.g. a service
// self-service-provisioning a secret for another service to pick up on its
// next bundle fetch. The latter path is scope-restricted: every file in the
// uploaded bundle must resolve to a namespace/service the caller is
// authorized to "put" to (see validateBundleScope), or nothing is written.
func (s *GitOpsServer) SyncBundle(stream adminv1.GitOpsService_SyncBundleServer) error {
	ctx := stream.Context()
	actor, scoped, err := s.authorizeGitOpsWrite(ctx, "SyncBundle")
	if err != nil {
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

	// A SPIFFE-authenticated (non-admin) caller may only push files that
	// resolve to a namespace/service it's authorized to write to — checked
	// against every file before anything is written, so a bundle that's
	// partly in-scope and partly not is rejected outright rather than
	// partially applied.
	if scoped {
		if err := s.validateBundleScope(ctx, tmpDir, secretsPath, configPath, actor); err != nil {
			return err
		}
	}

	// Run the SOPS decrypt + store pass. repoID is "" — a pushed bundle has
	// no registered git repository to attribute secrets to or diff deletions
	// against (see SyncFromDir's doc comment).
	result, err := s.syncer.SyncFromDir(ctx, tmpDir, secretsPath, headSHA, "", actor)
	if err != nil {
		return status.Errorf(codes.Internal, "sync: %v", err)
	}

	// Run the plain YAML config pass if a config_path was provided.
	configCount, _, configSkipped, configErr := s.syncer.SyncConfigFromDir(ctx, tmpDir, configPath, "", actor)
	if configErr != nil {
		slog.Warn("bundle config sync error", "err", configErr)
	}

	result.Skipped = append(result.Skipped, configSkipped...)

	return stream.SendAndClose(&adminv1.SyncBundleResponse{
		SecretsAdded:   int32(result.Added),
		SecretsUpdated: int32(result.Updated),
		SecretsDeleted: int32(result.Deleted),
		SyncSha:        result.SHA,
		ConfigsSynced:  int32(configCount),
		Errors:         skippedToErrors(result.Skipped),
	})
}

// PatchServiceConfig atomically applies an RFC 6902 JSON Patch to a
// service's existing plain config document (bytepunx/signet#38). Unlike
// SyncBundle's config path — a full-document replace, which would silently
// drop every field/entry a partial push doesn't mention — this only touches
// the paths the patch actually names, applied server-side inside a single
// transaction (store.Store.PatchServiceConfig) so there's no client-visible
// read step for a concurrent unrelated change to race against.
//
// Reachable the same two ways as SyncBundle: an admin bearer token (full
// access), or a workload's own SPIFFE mTLS identity, scoped via checker.Allow
// with the "put" permission — the same permission and exact-match convention
// SyncBundle's validateBundleScope already uses, so a workload patching its
// own namespace/service needs no policy row, and anything broader needs an
// explicit policy grant exactly like a cross-service push would.
func (s *GitOpsServer) PatchServiceConfig(ctx context.Context, req *adminv1.PatchServiceConfigRequest) (*adminv1.PatchServiceConfigResponse, error) {
	actor, scoped, err := s.authorizeGitOpsWrite(ctx, "PatchServiceConfig")
	if err != nil {
		return nil, err
	}
	if req.GetNamespace() == "" || req.GetService() == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace and service are required")
	}
	if len(req.GetOperations()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "operations must not be empty")
	}
	if scoped {
		if err := s.checker.Allow(ctx, actor, "put", req.GetNamespace(), req.GetService(), ""); err != nil {
			return nil, toGRPCError(err)
		}
	}

	patch, err := decodeJSONPatch(req.GetOperations())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "decode patch: %v", err)
	}

	version, err := s.store.PatchServiceConfig(ctx, req.GetNamespace(), req.GetService(), func(current json.RawMessage) (json.RawMessage, error) {
		patched, err := patch.Apply(current)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errPatchApplyFailed, err)
		}
		return patched, nil
	})
	if err != nil {
		if errors.Is(err, errPatchApplyFailed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, toGRPCError(err)
	}

	if s.audit != nil {
		if auditErr := s.audit.Record(ctx, audit.Entry{
			SPIFFEID:   actor,
			Action:     "patch_config",
			Namespace:  req.GetNamespace(),
			SecretName: req.GetService() + "/" + configAuditName,
			Outcome:    "permitted",
		}); auditErr != nil {
			slog.Error("audit write failed for PatchServiceConfig", "namespace", req.GetNamespace(), "service", req.GetService(), "err", auditErr)
		}
	}
	if s.bus != nil {
		s.bus.NotifyService(req.GetNamespace(), req.GetService())
		s.bus.NotifyBundle(req.GetNamespace(), req.GetService())
	}

	return &adminv1.PatchServiceConfigResponse{Version: int32(version)}, nil
}

// errPatchApplyFailed distinguishes a client-supplied-patch problem (bad
// path, failed "test" precondition, malformed op) from a store/database
// error inside PatchServiceConfig's apply closure — the two need different
// gRPC codes (InvalidArgument vs. whatever toGRPCError derives).
var errPatchApplyFailed = errors.New("apply patch")

// decodeJSONPatch converts the request's typed operations into an RFC 6902
// JSON Patch document. Proto doesn't have a native "arbitrary JSON value"
// type for Value's omitted case (a "remove"/"move"/"copy" operation has no
// value), so an unset Value is encoded as JSON null via structpb's own
// nil-safety rather than omitted entirely — evanphx/json-patch ignores
// "value" for operations that don't use it, so this is harmless.
func decodeJSONPatch(ops []*adminv1.JsonPatchOperation) (jsonpatch.Patch, error) {
	raw := make([]map[string]any, len(ops))
	for i, op := range ops {
		entry := map[string]any{
			"op":   op.GetOp(),
			"path": op.GetPath(),
		}
		if op.GetFrom() != "" {
			entry["from"] = op.GetFrom()
		}
		if op.GetValue() != nil {
			entry["value"] = op.GetValue().AsInterface()
		}
		raw[i] = entry
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encode operations: %w", err)
	}
	return jsonpatch.DecodePatch(encoded)
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
