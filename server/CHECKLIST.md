# signet implementation checklist

## Completed

- [x] `internal/crypto` — AES-256-GCM encryption, KeyStore with memguard (mlock, guard pages)
- [x] `internal/unseal` — Manager state machine, direct key, Shamir shares (GF(2^8)), TPM stub
- [x] `internal/store` — CockroachDB/pgx storage, embedded migrations, secret/policy/audit CRUD
- [x] Proto codegen — `buf generate`; Go stubs in `gen/` for `SecretsService` and `AdminService`/`GitOpsService`; proto files updated with `service` field and correct response type names
- [x] `internal/audit` — HMAC-SHA256 chain writer, `Zero()` on seal, chain head loaded from DB on startup

### `internal/auth`
- [x] `SPIFFEIDFromContext` — extracts and validates SPIFFE URI SAN from verified mTLS peer cert in gRPC context
- [x] `evalPolicies` / `Checker.Allow` — glob-matches `namespace/secretName` against store policies; `*` permission wildcard; skips malformed patterns
- [x] `TokenFromMetadata` — extracts `Bearer <token>` from gRPC incoming metadata for admin endpoint
- [x] `TokenValidator.Validate` — Kubernetes TokenReview API via `k8s.io/client-go`
- [x] Add `github.com/spiffe/go-spiffe/v2` and `k8s.io/client-go` dependencies

### `internal/api`
- [x] `SecretsService.GetSecret` — auth interceptor → policy check → `store.GetSecret` → unwrap DEK → decrypt → return; audit every path (permitted and denied)
- [x] `SecretsService.WatchSecret` — server-streaming; subscribes to Bus before initial fetch so no notification is lost; re-evaluates + re-fetches on push; sends EVENT_TYPE_DELETED before closing when secret is removed
- [x] `AdminService.UnsealKey` — SA token auth → `unseal.Manager.UnsealWithKey`
- [x] `AdminService.UnsealShare` — SA token auth → `unseal.Manager.SubmitShare`
- [x] `AdminService.Seal` — SA token auth → `unseal.Manager.Seal`
- [x] `AdminService.Status` — SA token auth → `unseal.Manager.Status`
- [x] `GitOpsService` — all 8 methods fully implemented (GetSOPSPublicKey, RotateSOPSKey, ListSOPSKeys, PruneSOPSKey, RegisterRepository, ListRepositories, RemoveRepository, TriggerSync)
- [x] `internal/api/bus.go` — in-process pub/sub Bus; buffered channels coalesce rapid notifications; null-byte key separator prevents collisions
- [x] `internal/api/errors.go` — maps all internal sentinel errors to appropriate gRPC status codes; unknown errors map to Internal without leaking detail
- [x] `internal/api/webhook.go` — HTTP handler for `POST /webhook/github/{repo_id}`; body size-limited; HMAC signature verified; push events dispatched to Syncer

### `internal/server`
- [x] gRPC server setup with go-spiffe mTLS credentials (`grpccredentials.MTLSServerCredentials`) for the workload listener; `spiffetls.AuthorizeMemberOf` accepts only SVIDs in the configured trust domain
- [x] Separate admin listener (plain TCP, no TLS; port-forward only in production)
- [x] Optional HTTP webhook listener on configurable port (`:8445`); managed alongside gRPC servers in Run/drain
- [x] Graceful shutdown: `drain()` runs `GracefulStop` in parallel on all servers; force-stops after `DrainTimeout` (default 30s); closes X509Source; then calls `mgr.Seal()`
- [x] Panic recovery interceptors on both unary and stream handlers — converts panics to `codes.Internal` so one bad request cannot crash the server
- [x] `SpireSource` / `SpireCredentials` / `NewFromSPIRE` helpers for production wiring
- [x] `server.New` accepts service interfaces so tests can inject stubs without needing SPIRE or a database

### `cmd/signetd`
- [x] Config loading via CLI flags + `SIGNET_*` env vars (flags take precedence); no viper dependency needed
- [x] Wire `store.Store`, `unseal.Manager`, `audit.Writer`, `api` handlers, `server.NewFromSPIRE`
- [x] Refuse to start if config invalid; all validation errors reported together (not one at a time)
- [x] Startup log emits addresses, trust domain, seal state, and Shamir shape — never DB connection strings or key material
- [x] `signal.NotifyContext` for SIGTERM/SIGINT; `audit.Writer.Zero()` and `store.Close()` deferred for clean shutdown
- [x] `buildK8sClient()` uses in-cluster service account; clear error if not running in Kubernetes
- [x] `WebhookAddr` / `WebhookBaseURL` config fields wired through; `gitops.Syncer` and `gitops.Reconciler` constructed and started

### `cmd/signet` (operator CLI)
- [x] Add `github.com/spf13/cobra` dependency
- [x] `signet unseal key --key-file <path>` — read key, call `AdminService.UnsealKey`
- [x] `signet unseal share --share <hex>` / `--share-file <path>` — call `AdminService.UnsealShare`
- [x] `signet seal` — call `AdminService.Seal`
- [x] `signet status` — call `AdminService.Status`, print state + shares received/required
- [x] Config subcommand: `signet config set server <url>` / `signet config set token-file <path>` (persist to `~/.config/signet/config.yaml`)
- [x] `--server`, `--token`, `--token-file` persistent flags; token resolution: flag → flag-file → config-file → error
- [x] Insecure gRPC credentials (admin is localhost port-forward only); `PerRPCCredentials` injects `Authorization: Bearer` on every call
- [x] `signet sops-key get/rotate/list/prune` — age key lifecycle management
- [x] `signet repo add/list/remove/sync` — repository registration and manual sync

### `deploy/`
- [x] Helm chart: signetd Deployment, Service (workload port 8443; webhook port 8445; 8444 is localhost-only), ConfigMap, Secret placeholder
- [x] CockroachDB StatefulSet (insecure, single-node dev mode) gated behind `cockroachdb.enabled`; headless Service; postStart lifecycle hook creates `signet` database
- [x] NetworkPolicy: deny all ingress to CockroachDB except from signetd pods; ingress on 8443 (gRPC) and 8445 (webhook) to signetd; egress restricted to DNS + CockroachDB + k8s API
- [x] RBAC: `signet` ServiceAccount with ClusterRole for TokenReview creation; `signet-admin` ServiceAccount for operator token issuance (`kubectl create token`)
- [x] SPIRE `ClusterSpiffeID` for signetd workload identity; example workload registration commented in the same file
- [x] `deploy/spire/` — SPIRE server ConfigMap (k8s_psat attestor, SQLite, disk key manager, k8sbundle notifier) and agent ConfigMap (k8s_psat + k8s workload attestor)

### GitOps / SOPS integration (GitHub)
- [x] Add `filippo.io/age`, `github.com/getsops/sops/v3`, `github.com/go-git/go-git/v5`, `github.com/google/go-github/v68` dependencies
- [x] Migration `000002_gitops.sql` — `sops_age_keys` and `git_repositories` tables
- [x] `store/sops.go` — `PutSOPSKey`, `GetSOPSKey`, `GetActiveSOPSKey`, `ListSOPSKeys`, `DeactivateSOPSKey`, `DeleteSOPSKey`
- [x] `store/repos.go` — `PutRepository`, `GetRepository`, `GetRepositoryByName`, `ListRepositories`, `DeleteRepository`, `UpdateSyncState`
- [x] `internal/gitops/keys.go` — `GenerateAgeKey` / `DecryptAgeKey` using `keyUnwrapper` interface; private key encrypted under master key; zeroed after use
- [x] `internal/gitops/sops.go` — `DecryptFile` — in-memory SOPS decryption; age identities injected via `ParsedIdentities.ApplyToMasterKey`; extracts top-level `value` field; no plaintext written to disk
- [x] `internal/gitops/path.go` — `ParseSecretPath` — maps `<root>/<ns>/<svc>/<name>.yaml` to store tuple; rejects traversal and out-of-root paths
- [x] `internal/gitops/github.go` — `VerifyWebhookSignature` (HMAC-SHA256, constant-time); `ParsePushEvent`; `BranchFromRef`; `ChangedFiles`
- [x] `internal/gitops/sync.go` — `Syncer` with `SyncFromPush` (incremental) and `FullSync` (full clone); SSH deploy key auth via go-git; per-secret DEK re-encryption under master key
- [x] `internal/gitops/reconcile.go` — `Reconciler` with configurable periodic full-sync loop; immediate sync on first `Run`
- [x] `internal/gitops/path_test.go`, `github_test.go` — unit tests for all pure functions

---

### Reconciler lifecycle (unseal-aware)
- [x] Add `StatusCh() <-chan struct{}` to `unseal.Manager` — capacity-1 ping channel; non-blocking send on every state transition; consumers call `Status()` to read current state (coalesces rapid transitions)
- [x] Update `cmd/signetd/main.go` — `runReconcilerLifecycle` goroutine watches `StatusCh()`; starts reconciler on unseal, cancels on seal; restarts on re-unseal
- [x] `WebhookHandler` — checks `sealChecker.Status()` before processing; returns HTTP 503 + `Retry-After: 30` when sealed so GitHub retries automatically

### Documentation
- [x] `deploy/docs/sops-setup.md` — keypair generation, `.sops.yaml` config, file format + path convention, encryption steps, key rotation walkthrough
- [x] `deploy/docs/github-webhook-setup.md` — deploy key setup, `signet repo add`, GitHub webhook configuration, initial sync, sealed-state retry behaviour, troubleshooting table

---

## Open

### CI / GitHub Actions

#### Test and lint (runs on every PR and push to main)
- [x] `.github/workflows/ci.yml` — test, lint, proto check, helm lint, binary build, Docker build (no push) jobs
- [x] `.golangci.yml` — `errcheck`, `govet`, `staticcheck`, `gosec`, `misspell`, `unused`, `unparam`, `gocritic`, `nilerr`; gen/ excluded; test files relaxed
- [x] Proto codegen check — `buf generate` + `git diff --exit-code gen/` in CI
- [x] `helm lint` + `helm template` steps in CI to catch chart regressions

#### Release workflow (triggered on `v*` tag push)
- [x] `.github/workflows/release.yml` — matrix build of `signet` CLI for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`; `sha256sums.txt` attached to GitHub release
- [x] `signetd` Docker image built for `linux/amd64` + `linux/arm64`, pushed to `ghcr.io/bytepunx/signetd` with semver + `latest` tags
- [x] Helm chart packaged and pushed as OCI artifact to `ghcr.io/bytepunx/charts/signet` on release
- [x] Release notes from `CHANGELOG.md` if present, falling back to `git log --oneline` since previous tag

#### Docker image hardening
- [x] `Dockerfile` — multi-stage: `golang:1.26-alpine` builder → `gcr.io/distroless/static-debian12:nonroot` runtime; both `signetd` and `signet` targets; `CGO_ENABLED=0`, `-trimpath`, `-ldflags="-s -w"`

---

### Helm chart — installation UX

#### Values and validation
- [x] `deploy/helm/signet/values.schema.json` — JSON Schema (draft-07) for all chart values; `signet.trustDomain` marked `required` with `minLength: 1` and a description explaining the requirement; bad types fail `helm install` before rendering
- [x] `global.image.registry` override — `image.registry` split from `image.repository`; `signet.image` helper in `_helpers.tpl` applies the global override; `deployment.yaml` uses the helper; air-gapped installs only need to set one value

#### Ingress for the webhook endpoint
- [x] `templates/ingress.yaml` — gated on `ingress.enabled`; supports `className`, `annotations`, `hosts[].paths[]`, `tls`; routes to the `webhook` named port (8445)
- [x] `ingress.*` section in `values.yaml` with commented nginx-ingress and cert-manager annotation examples

#### Operational resources
- [x] `templates/poddisruptionbudget.yaml` — `minAvailable: 1`; only rendered when `replicaCount > 1`
- [x] `templates/hpa.yaml` — gated on `autoscaling.enabled` (default false); includes scaling caveat in values comment
- [x] `templates/tests/test-connection.yaml` — `helm test` hook; `busybox:1.36` pod; `nc -z` probe against gRPC port; cleaned up on success

#### SPIRE dependency documentation
- [x] `deploy/helm/signet/README.md` — prerequisites table, quick-start `helm install` one-liner, required values table, full values reference, production checklist, scaling caveat, air-gapped install example, `helm test` note

---

### Documentation — user-facing

#### Getting started guide
- [x] `docs/getting-started.md` — end-to-end walkthrough: SPIRE deploy → signet Helm install → unseal (direct-key and Shamir) → SPIRE entry → policy → encrypt + push secret → workload retrieval

#### Installation and configuration reference
- [x] `docs/installation.md` — complete Helm values table (all values, defaults, required column); env var reference for running outside Kubernetes; SPIRE ClusterSpiffeID examples; Shamir unseal walkthrough; scaling caveats; CockroachDB production sizing table

#### CLI reference
- [x] `docs/cli.md` — full command reference: `config set`, `status`, `unseal key`, `unseal share`, `seal`, `sops-key get/rotate/list/prune`, `repo add/list/remove/sync`; all flags documented with examples and sample output

#### Working with SOPS-encrypted secrets (user guide)
- [x] `docs/secrets-workflow.md` — four-part guide: initial setup (age key, `.sops.yaml`, deploy key, repo registration) → create + encrypt secrets (path convention, file format, worked example) → verify retrieval (Go client snippet) → key rotation (rotate → re-encrypt → prune); troubleshooting table

#### Access policy guide
- [x] `docs/policies.md` — SPIFFE ID glob syntax with match/no-match table; `signet policy create/list/remove` with examples; cross-namespace, wildcard, and short-lived operator patterns; audit log query; least-privilege design guidance

---

## Open

### `signet init` — cluster-native bootstrap and unseal

Implements Mechanism 3 (Kubernetes Secret) from Section 6 of the design. A single command
that prepares the cluster for single-key operation and unseals signet. Reference: `design/draft.md §6`.

#### Dependencies

- [x] Add `k8s.io/client-go` to `go.mod` — already present at v0.36.2
- [x] Add `k8s.io/api` and `k8s.io/apimachinery` at matching versions — already present at v0.36.2

#### `signet init` CLI command (`cmd/signet/init_cmd.go`)

- [x] Implement `signet init` cobra command with flags: `--namespace` (default `signet`), `--key-secret` (default `signet-master-key`), `--kube-context`, `--force`; uses `--server` and `--token` from root
- [x] Kubeconfig discovery: `$KUBECONFIG` → `~/.kube/config` → in-cluster SA; `--kube-context` overrides the active context without mutating the kubeconfig file on disk
- [x] Check seal state before doing any work; print current state and exit 0 if already unsealed
- [x] Secret read path: if `--key-secret` exists in `--namespace`, read `data["master.key"]`; fail clearly if the field is missing
- [x] Secret create path: if Secret does not exist (or `--force`), generate 32 bytes via `crypto/rand`; create or patch the Secret; print a warning that the Secret contains the master key and should be access-controlled
- [x] Call admin API `UnsealKey` with key bytes; handle `--token` auth same as other commands
- [x] Verify post-unseal state via `Status` RPC; print final state summary
- [x] `--dry-run` flag: prints every action that would be taken (Secret create/read, unseal call) without executing any of them; useful for CI validation

#### Tests for `signet init`

- [x] Unit test: Secret create path — mock Kubernetes client + mock admin gRPC server; verify Secret is created with correct field and unseal is called with those bytes
- [x] Unit test: Secret read path — pre-existing Secret; verify key is read (not regenerated) and unseal is called
- [x] Unit test: `--force` flag — pre-existing Secret; verify Secret is overwritten with new key
- [x] Unit test: already-unsealed guard — verify command exits 0 without touching the Secret or calling UnsealKey
- [x] Unit test: `--dry-run` — verify no Kubernetes API writes and no gRPC calls are made

#### signetd auto-unseal (`cmd/signetd/kube_unseal.go`)

- [x] New config field `KubeUnsealSecret string` (env: `SIGNET_KUBE_UNSEAL_SECRET`); added to `cmd/signetd/config.go`
- [x] In `cmd/signetd/main.go`: before listeners start, if `KubeUnsealSecret` is set call `attemptKubeUnseal` synchronously with a 30-second timeout; non-fatal on failure
- [x] `attemptKubeUnseal`: builds an in-cluster Kubernetes client; reads pod namespace from SA token file; fetches the named Secret; reads `data["master.key"]`; calls `unsealMgr.UnsealWithKey(key)`; key zeroed by `KeyStore.Set`
- [x] Graceful degradation: missing Secret, RBAC denial, or unreachable API all log WARN and return — server stays sealed
- [x] Unit tests for `attemptKubeUnseal`: valid Secret → unsealed; missing field → sealed; Secret not found → sealed; already unsealed → no-op (`cmd/signetd/kube_unseal_test.go`)

#### Helm chart updates

- [x] `deploy/helm/signet/templates/role.yaml` — namespaced Role; `resourceNames` scoped to the configured Secret name; only rendered when `autoUnseal.enabled`
- [x] `deploy/helm/signet/templates/rolebinding.yaml` — binds Role to signetd ServiceAccount when `autoUnseal.enabled`
- [x] Added `autoUnseal.enabled: false` and `autoUnseal.secretName: "signet-master-key"` to `values.yaml`
- [x] Added `autoUnseal.*` to `values.schema.json` with description noting the trust-boundary trade-off
- [x] `templates/configmap.yaml` emits `SIGNET_KUBE_UNSEAL_SECRET` when `autoUnseal.enabled`
- [x] Updated `deploy/helm/signet/README.md` with Auto-unseal section covering the trust boundary trade-off

#### Documentation

- [x] Added `signet init` to `docs/cli.md` — synopsis, flags table, step-by-step output examples, `--force` and `--dry-run` semantics
- [x] Added "signet init — recommended for dev clusters" to `docs/getting-started.md` with Shamir as the production alternative
- [x] Added "autoUnseal.* — Auto-unseal via Kubernetes Secret" section to `docs/installation.md` with when-to-use, Helm values, and combined workflow

---

### `signet bundle push` — local-repo-to-signet upload

Bootstrap signet secrets from a local git repository without pushing to a remote.
SOPS-encrypted YAML files are packaged by the CLI, streamed to the server as a
tar.gz, and decrypted server-side — operator machines never handle plaintext.

#### Proto (`proto/admin/v1/admin.proto`)

- [x] Added `rpc SyncBundle(stream SyncBundleChunk) returns (SyncBundleResponse)` to `GitOpsService`
- [x] `SyncBundleChunk` with `oneof payload { SyncBundleHeader header = 1; bytes data = 2; }`
- [x] `SyncBundleHeader { string secrets_path = 1; string head_sha = 2; }`
- [x] `SyncBundleResponse { int32 secrets_added/updated/deleted; string sync_sha; repeated string errors; }`
- [x] Ran `buf generate` — all types in `gen/admin/v1/`

#### Server — gitops sync refactor (`internal/gitops/sync.go`)

- [x] Extracted `SyncFromDir(ctx, dir, secretsPath, headSHA string) (*SyncResult, error)` from `FullSync` walk body
- [x] `FullSync` now delegates to `SyncFromDir` after cloning — no duplicated logic

#### Server — archive extraction (`internal/api/bundle.go`)

- [x] `extractTarGz(r io.Reader, dir string) error` — decompresses tar.gz, skips symlinks/devices
- [x] Path traversal protection: rejects `..` components and any path that escapes the extraction dir
- [x] Size cap: `maxBundleSize = 256 MiB`; accumulates across all files, rejects at overflow

#### Server — gRPC handler (`internal/api/gitops.go`)

- [x] `GitOpsServer.SyncBundle(stream adminv1.GitOpsService_SyncBundleServer) error`
- [x] Auth: `requireToken` on `stream.Context()`
- [x] First chunk validated as `SyncBundleHeader`; data chunks accumulated
- [x] Extracts to temp dir via `extractTarGz`; calls `syncer.SyncFromDir`; returns `SyncBundleResponse`

#### CLI (`cmd/signet/bundle_cmd.go`)

- [x] `signet bundle push [path]` cobra subcommand under `bundle`
- [x] Opens repo with `gogit.PlainOpen`; resolves HEAD commit; iterates tree files under `--secrets-path`
- [x] Includes only `.yaml` files tracked by git (HEAD commit tree, not filesystem walk)
- [x] Builds tar.gz in memory with `archive/tar` + `compress/gzip`; streams in 64 KiB chunks
- [x] Sends `SyncBundleChunk_Header` first, then `SyncBundleChunk_Data` chunks
- [x] Calls `stream.CloseAndRecv()` and prints added/updated/deleted/sha summary
- [x] `--secrets-path` flag (default `secrets/`)

#### Tests

- [x] `internal/api/bundle_test.go` — `extractTarGz`: basic extraction; path traversal rejection; symlink skip; size limit
- [x] `internal/gitops/sync_test.go` — `SyncFromDir`: no SOPS keys error; missing secrets dir error; headSHA passed through contract

---

## Future / not yet specified
<!-- add items here as we discuss additional features -->
