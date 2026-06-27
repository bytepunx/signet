# Kubernetes-Native Secret Store: Architecture Decision Summary

## Context

This document summarizes the key design decisions for a purpose-built secret store targeting
Kubernetes environments, including air-gapped and on-premises deployments. The retrieval and
injection of secrets into pods/containers is handled by a separate mechanism (CSI driver or
in-process bootstrapper); this store is concerned only with secure storage, access control,
and the API surface through which secrets are retrieved.

---

## 1. Why Not Kubernetes Secrets

Kubernetes Secrets are unsuitable as a foundation because:

- Stored as base64 (not encrypted) in etcd by default
- Encryption at rest requires explicit configuration and is frequently misconfigured
- Environment variable injection leaks secrets through `/proc`, logs, and crash dumps
- RBAC is namespace-scoped with no field-level access control — `list` on Secrets exposes all of them
- No built-in rotation, versioning, expiry, or audit trail
- Node-level blast radius: a compromised node exposes all secrets for co-located pods

---

## 2. Storage Backend

**Decision: CockroachDB**, deployed as a StatefulSet in the secret manager namespace.

Rationale:
- Native encryption at rest (AES-256)
- Raft-based replication provides HA across node failures without a separate replication implementation
- SQL schema enables versioning, expiry, and structured audit logging without building those primitives
- Mature Kubernetes operator
- Single well-defined client (the secret manager API server) eliminates noisy-neighbor concerns
- Alternatives considered:

| Store | Reason Not Selected |
|---|---|
| etcd (dedicated) | No field-level access control, manual versioning/audit, operational overhead of separate cluster |
| TiKV | Over-engineered for secret-store scale; requires separate PD cluster |
| Consul OSS | No application-level encryption at rest in OSS tier |
| FoundationDB | Immature Kubernetes operational story |
| CockroachDB / Yugabyte | CockroachDB selected over Yugabyte for more mature k8s operator |

### Schema Design

```sql
secrets (namespace, service, secret_name, version, encrypted_dek, ciphertext, expires_at, metadata)
access_policies (spiffe_id, namespace, secret_pattern, permissions)
audit_log (ts, spiffe_id, action, namespace, secret_name, outcome, peer_ip)  -- TTL: 90 days
```

---

## 3. Network Isolation

**Decision: NetworkPolicy + RBAC to isolate the storage layer within the secret manager namespace.**

- No pod outside the namespace can open a TCP connection to CockroachDB
- No service account outside the namespace has RBAC access to storage pods
- The secret manager API server is the sole egress point for secrets
- CockroachDB inter-node traffic is TLS-encrypted, preventing node-level sniffing

This does not protect against cluster-admin compromise — that is the accepted trust boundary.

---

## 4. Client Authentication: mTLS via SPIFFE/SPIRE

**Decision: mTLS with SPIFFE SVIDs as the authentication mechanism at the API server.**

SPIFFE (Secure Production Identity Framework for Everyone) provides workload identity without long-lived credentials. A SPIFFE ID is a URI embedded as a Subject Alternative Name (SAN) in an X.509 certificate:

```
spiffe://cluster.local/ns/payments/sa/transaction-service
```

SPIRE (the reference implementation) issues these certificates (SVIDs) by:

1. **Node attestation** — verifying the node is a legitimate cluster member via kubelet credentials
2. **Workload attestation** — asking the kernel (not the workload) what process is requesting an SVID, based on PID/UID/pod/namespace/service account

SVIDs are short-lived (hours or minutes) and rotated automatically by the SPIRE agent. There is no long-lived credential to steal.

### Authorization Flow

```
Workload presents SVID over mTLS
  → API server verifies certificate chain to trusted SPIRE CA bundle
  → Extracts SPIFFE ID from verified URI SAN (e.g. spiffe://cluster.local/ns/payments/sa/api)
  → Exact-match convention: if SPIFFE ns+SA == secret namespace+service → permit immediately
  → Otherwise: query access_policies for SPIFFE ID → evaluate glob patterns
  → If permitted: decrypt DEK with master KEK, decrypt secret with DEK, return over mTLS channel
  → Write to audit_log regardless of outcome
```

### Policy Model

**Convention-first:** a workload whose Kubernetes identity (`ns/<namespace>/sa/<service-account>`)
exactly matches the secret's `namespace` and `service` fields is permitted automatically — no row
in `access_policies` is required. This covers the primary usage pattern without any operator
configuration. Explicit policies are only necessary for cross-service or cross-namespace access.

Policies support prefix/glob matching on secret paths, enabling coarse or fine-grained control:

```
# Entire namespace reads a secret class
spiffe://cluster.local/ns/payments/*  →  payments/*/db-read-replica-*

# Specific service, specific secret
spiffe://cluster.local/ns/payments/sa/tx-service  →  payments/tx-service/stripe-api-key

# Explicit cross-namespace grant
spiffe://cluster.local/ns/reporting/sa/etl-job  →  payments/shared/read-only-db-url
```

---

## 5. Envelope Encryption

**Decision: Per-secret DEKs wrapped by a KEK, wrapped by a master key held only in memory.**

```
Plaintext Secret
  → Encrypt with DEK (AES-256-GCM, random per secret)
  → Encrypted blob stored in CockroachDB

DEK
  → Encrypted with KEK (AES-256-GCM key wrapping)
  → Stored alongside encrypted blob

KEK
  → Encrypted with Master Key

Master Key
  → Never touches disk; held in locked memory (mlock)
  → Zeroed on shutdown
```

Properties:
- Compromising one secret does not compromise others
- KEK rotation re-wraps DEKs without re-encrypting all secret blobs
- Master key rotation re-wraps all KEKs
- `memguard` used for in-process key material protection (guard pages, canaries, zeroing on GC)

The API server enters a **sealed state** on startup — it can query CockroachDB but cannot
unwrap any DEK and will not serve any secret until the master key is loaded into memory via
one of the three unseal mechanisms described in Section 6.

---

## 6. Unsealing

The master key is never written to disk. On every restart the API server starts sealed and
requires an explicit unseal operation before serving secrets. Three mechanisms are supported;
operators choose one at deployment time.

### Mechanism 1: Direct Key

The operator holds the master key in their own secure storage (password manager, encrypted
file, hardware token) and provides it at startup via the signet CLI.

```bash
signet unseal key --key-file ./master.key
```

Appropriate for single-operator or development deployments. The security of the master key
is entirely the operator's responsibility.

### Mechanism 2: Shamir's Secret Sharing

The master key is split into N shares at initialisation, each distributed to a separate
keyholder. Any M shares reconstruct the master key (e.g. 3-of-5). Fewer than M shares are
information-theoretically zero — no amount of computation recovers the key from an incomplete
set.

Each keyholder authenticates independently using their own cluster credentials and submits
their share via the signet CLI. No credentials are shared between keyholders.

```bash
# Each keyholder independently:
kubectl create token signet-admin -n signet --duration=1h
kubectl port-forward -n signet svc/signet-admin 8443:8443
signet unseal share --share $MY_SHARE
```

The API server accumulates shares in memory. Accumulated shares are wiped after a configurable
timeout if the threshold is not reached. Once M shares are received the master key is
reconstructed in mlock'd memory and all share material is immediately zeroed.

The audit log records each share submission against the individual keyholder's SA token,
providing full attribution.

### Mechanism 3: Kubernetes Secret (cluster-native)

For single-operator or development deployments where the Kubernetes control plane is the
accepted trust boundary, the master key can be stored as a Kubernetes Secret and managed
entirely within the cluster.

#### Trust model

The Kubernetes Secret is protected by cluster RBAC. Anyone with `get` permission on that
Secret — cluster-admins and any namespace admins explicitly granted access — can read the
master key. This is weaker than Shamir (which requires M keyholders to collaborate) but
stronger than leaving the key in an env var or configmap. It is the appropriate choice when:

- The threat model accepts cluster-admin as the root of trust
- There is no multi-keyholder requirement
- Operational simplicity is the priority (dev clusters, small single-operator deployments)

It is NOT appropriate when cluster compromise must not imply master key disclosure.

#### Bootstrap: `signet init`

A single CLI command handles the entire cluster preparation and unseal workflow:

```
signet init [flags]
  --namespace      Kubernetes namespace where signet is deployed (default: signet)
  --key-secret     Name of the Kubernetes Secret to create/read (default: signet-master-key)
  --kube-context   kubectl context to use (default: current context)
  --server         signet admin endpoint (default: from config)
  --token          admin Bearer token
  --force          Regenerate key and overwrite existing Secret
```

Execution steps (idempotent):

```
1. Load kubeconfig; create Kubernetes API client
2. Check seal state via admin API
   → If already unsealed: print state and exit 0
3. Look for the key Secret in --namespace
   → If found and --force not set: use existing key bytes
   → If found and --force set: generate new 32-byte random key; overwrite Secret
   → If not found: generate new 32-byte random key; create Secret
4. Unseal signet via admin API with key bytes from Secret
5. Verify seal state is now unsealed
6. Print summary of what was created/reused and final state
```

The Secret format is:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: signet-master-key
  namespace: signet
type: Opaque
data:
  master.key: <base64-encoded 32 bytes>
```

`signet init` uses `client-go` directly (not a `kubectl` subprocess) so it works headlessly
in CI pipelines and scripts. Kubeconfig is discovered via the standard chain:
`$KUBECONFIG` → `~/.kube/config` → in-cluster service account (when running inside a pod).

#### Auto-unseal on restart

To eliminate manual re-unseal after pod restarts, signetd can be configured to fetch the
master key from the Secret at startup and unseal itself automatically:

```
SIGNET_KUBE_UNSEAL_SECRET=signet-master-key
```

When this env var is set and the server is sealed at startup, signetd:
1. Attempts to fetch the named Secret from its own namespace via the Kubernetes API
2. Reads the `master.key` field
3. Calls its own unseal path with the key bytes
4. Logs the outcome (unsealed or error)

If the Secret does not exist or the fetch fails (e.g., RBAC misconfiguration), signetd logs
the error and remains sealed — it does NOT fail fatally. Manual unseal remains possible.

This requires granting signetd's ServiceAccount `get` on Secrets in the signet namespace,
which the Helm chart manages via a dedicated Role + RoleBinding (not a ClusterRole, keeping
the permission namespace-scoped and minimal).

Auto-unseal is disabled by default. Enable via Helm:

```yaml
autoUnseal:
  enabled: true
  secretName: signet-master-key
```

#### Interaction with `signet init`

```
Initial setup (once):
  signet init          → generates key, creates Secret, unseals

After pod restart (if auto-unseal enabled):
  signetd startup      → reads Secret, unseals automatically

After pod restart (if auto-unseal disabled):
  signet init          → reads existing Secret (no --force), re-unseals
```

### Mechanism 4: TPM / vTPM Auto-Unseal (opt-in)

For environments where TPM 2.0 or a cloud-provider vTPM is available and configured, the
master key can be sealed to TPM PCR state at initialisation. On subsequent restarts the API
server automatically unseals without operator intervention.

TPM availability by environment:
- **Bare metal (post ~2018):** TPM 2.0 near-universal on enterprise hardware
- **AWS:** NitroTPM, opt-in on Nitro-based instance types
- **GCP:** vTPM via Shielded VMs, opt-in
- **Azure:** vTPM via Trusted Launch, opt-in
- **Edge / ARM boards (Raspberry Pi etc.):** not available without add-on hardware

Note: cloud vTPM security guarantees are weaker than physical TPM — the hypervisor controls
vTPM state. For most threat models this is acceptable.

If TPM is configured and unavailable at startup the server hard-fails rather than falling
back to manual unsealing. An explicit `--tpm-fallback` flag must be passed to permit manual
unsealing when TPM fails.

### Sealing

The API server can be returned to sealed state at any time:

```bash
signet seal
```

This immediately zeroes the master key from mlock'd memory. The server stops serving secrets
and requires a full unseal operation to resume. Used for security incidents or planned
maintenance.

---

## 7. Persistent Storage

**Decision: Local volumes (`local` StorageClass) co-located with secret store pods.**

Rationale:
- Storage traverses no network path that could be intercepted
- Network storage (Ceph/Rook) adds operational surface without benefit since Raft replicates
  at the application layer
- Application-layer envelope encryption (AES-256-GCM per secret) provides the primary
  protection at rest; CockroachDB's native AES-256-GCM encryption provides a second layer

```
Physical disk
  → Filesystem (ext4/xfs)
  → CockroachDB data files (AES-256-GCM encrypted at application layer)
  → Per-secret envelope encryption (DEK/KEK/Master Key hierarchy)
```

Operators may additionally configure LUKS full-disk encryption on the underlying partition
as a further defense-in-depth layer. This is not required by signet and key management for
LUKS is the operator's responsibility.

---

## 8. Node Identity

**Decision: SPIRE with Kubernetes node attestation (kubelet credential verification) as the
baseline, requiring no additional hardware.**

Node attestation establishes that a SPIRE agent is running on a legitimate cluster node before
workload attestation is trusted. SPIRE's Kubernetes attestor plugin handles this with no
additional hardware:

- **Kubernetes attestor** — verifies node via Kubernetes API server (required baseline)

Environments with TPM 2.0 available may additionally configure the SPIRE TPM attestor for
hardware-rooted node identity, but this is not required.

---

## 9. PKI / Certificate Authority

**Decision: cert-manager + Smallstep Step CA for in-cluster PKI.**

- Step CA issues short-lived certificates with automated rotation, designed for high-volume
  automated environments
- Root CA private key generated offline and stored in encrypted cold storage
- Only intermediate CAs run in-cluster
- cert-manager handles certificate lifecycle for all in-cluster TLS

SPIRE is deployed before signet and serves as the primary source of workload identity (SVIDs)
for all service-to-service communication including signet's own API endpoints. This eliminates
the bootstrapping dependency between signet and cluster TLS infrastructure.

---

## 10. Bootstrap and Operator Access

SPIRE is deployed before signet. This resolves the bootstrapping problem — SPIRE uses the
Kubernetes API server and kubelet for attestation and has no dependency on signet. Once SPIRE
is running it immediately issues SVIDs to attested workloads including signet itself.

### Deployment Order

1. Deploy SPIRE (server + agents)
2. Deploy cert-manager + Step CA
3. Deploy signet — starts in sealed state; SPIRE immediately attests it and issues SVID
4. Operators perform unseal via CLI (Section 6):
   - Single-operator / dev: `signet init` (generates key, stores in Kubernetes Secret, unseals)
   - Multi-operator / production: `signet unseal share` per keyholder (Shamir)
5. Signet becomes operational; all subsequent workload access uses SPIFFE mTLS

### Operator CLI Access During Unseal

The signet admin endpoint is not exposed outside the cluster. Operators reach it via
`kubectl port-forward`, using their existing kubeconfig credentials to authenticate the
tunnel. Each operator authenticates independently at the application layer with a short-lived
Kubernetes ServiceAccount token — no credentials are shared between operators.

```bash
# Each operator independently:
kubectl create token signet-admin -n signet --duration=1h   # expires automatically
kubectl port-forward -n signet svc/signet-admin 8443:8443   # tunnel via kubeconfig creds
signet config set server https://localhost:8443
signet unseal share --share $MY_SHARE                       # or 'key' for mechanism 1
```

The bootstrap SA token is revoked once unsealing is complete. All subsequent admin operations
require a fresh short-lived token and an active port-forward session.

### Local Development

For local clusters (kind, minikube, k3s), the admin endpoint may be configured to accept
connections on loopback without TLS using `--dev-mode`. This mode refuses to start if the
bind address is not loopback and must never be used in production deployments.

---

## 11. Audit Logging

**Decision: Structured append-only logs forwarded out-of-cluster in near-real-time, with
HMAC chaining for tamper detection.**

- Every secret access (permitted or denied) written to `audit_log` table and forwarded
- HMAC chaining: each log entry includes an HMAC of the previous entry — retroactive tampering
  is detectable
- Log destination: out-of-cluster Loki or syslog-over-TLS to a separate host
- CockroachDB TTL on `audit_log` table handles retention automatically (default: 90 days)

---

## 12. GitOps / SOPS Integration

**Decision: Git repositories as the source of truth for secrets; SOPS + age for encryption at rest in the repository.**

Operators do not write secrets directly into signet via the API. Instead, secrets live in a git repository encrypted with SOPS and are pulled into signet automatically. This gives full git history, PR review, and branch-based promotion for secret changes.

### Why age (not PGP or KMS)

- age keys are short strings (bech32-encoded X25519) with no keyring daemon or external service dependency
- Key rotation is a two-step: generate new key → re-encrypt files → prune old key; no KMS account required
- Works in air-gapped environments with no cloud provider dependency
- signet generates and stores the age keypair itself; the private key is encrypted under the signet master key and never touches disk in plaintext

### Repository Secret Format

Each secret is a SOPS-encrypted YAML file with a single `value` key:

```yaml
# secrets/payments/api/stripe-key.yaml  (SOPS-encrypted)
value: sk_live_...
sops:
  age:
    - recipient: age1abc...   # signet's active public key
      enc: |
        ...
```

Path convention (single-environment): `<secrets_root>/<namespace>/<service>/<name>.yaml`

Path convention (multi-environment): `<secrets_root>/<env>/<namespace>/<service>/<name>.yaml`

The environment prefix is stripped at sync time; the decrypted `value` field is stored as `(namespace=payments, service=api, name=stripe-key)` regardless of which environment subdirectory it came from.

### Key Lifecycle

```
signet sops-key rotate          → generates X25519 keypair; stores encrypted privkey in DB
                                  deactivates previous key (kept for decryption)
signet sops-key get             → prints active age public key for .sops.yaml
# operator re-encrypts files in repo with new key
signet sops-key prune <pubkey>  → permanently deletes inactive key; blocked if still active
```

Multiple inactive keys are retained until pruned so that any files still encrypted to the old key can still be synced. signet tries all stored keys during SOPS decryption.

### Environment Separation

**Decision: per-environment age keys; mono-repo with environment subdirectories.**

When operating multiple clusters (prod, staging, dev), each signetd instance is given a distinct environment label via `SIGNET_ENVIRONMENT`. This label:

1. Is stored on age keys at rotation time (`sops_age_keys.environment` column).
2. Filters which keys are loaded for decryption — only keys matching the instance's environment, or global keys (empty environment), are considered.
3. Is set as the environment on newly rotated keys, so the key cannot be mistakenly used by another environment's instance.

**Storage:** A single `sops_age_keys` table holds keys for all environments. The `environment` column is indexed; a `WHERE environment = $1 OR environment = ''` predicate isolates keys per instance. Global keys (created before `SIGNET_ENVIRONMENT` was set) remain usable by any instance.

**Repository layout:**

```
secrets/
├── prod/
│   └── payments/api/stripe-key.yaml   # encrypted with prod age key
├── staging/
│   └── payments/api/stripe-key.yaml   # encrypted with staging age key
└── dev/
    └── payments/api/stripe-key.yaml   # encrypted with dev age key
```

Each instance registers the repository with `--secrets-path secrets/<env>/` so that it only processes its own subdirectory.

**`.sops.yaml` with per-environment rules** (generated by `signet sops-key update-config`):

```yaml
# signet-specific: maps environment names to their active age public keys.
# Ignored by the sops CLI; used by signet tooling for update-config idempotency.
environments:
  prod:    age1prod111...
  staging: age1staging222...
  dev:     age1dev333...

# Standard SOPS creation rules (used by the sops CLI for encryption).
# signet_environment is ignored by sops; it lets update-config find the right
# rule without touching others.
creation_rules:
  - signet_environment: prod
    path_regex: ^secrets/prod/
    age: age1prod111...
  - signet_environment: staging
    path_regex: ^secrets/staging/
    age: age1staging222...
  - signet_environment: dev
    path_regex: ^secrets/dev/
    age: age1dev333...
```

A prod signetd instance cannot decrypt staging or dev secrets even if it has access to the same repository — the SOPS ciphertext is encrypted to a different public key, and that private key is not in the prod instance's key store.

**Backward compatibility:** `SIGNET_ENVIRONMENT` defaults to empty. Existing deployments continue to work without change — global keys are returned by `GetActiveSOPSKey` and `ListSOPSKeys` when no environment is set.

### Repository Registration

```
signet repo add --name infra-secrets \
                --repo-url git@github.com:org/infra-secrets \
                --branch main \
                --secrets-path secrets/ \
                --deploy-key ./deploy_key
```

signet encrypts the SSH deploy key under the master key and stores it. It generates a random webhook secret and returns the GitHub webhook URL and secret once — not retrievable again.

### Sync Flow

**On push (webhook-driven):**

```
GitHub push event
  → POST /webhook/github/{repo_id}
  → Verify HMAC-SHA256 signature (constant-time)
  → Check branch matches tracked branch
  → Parse changed/deleted file list from push event
  → For each changed .yaml file under secrets_path:
      clone repo at headSHA (ephemeral temp dir, SSH deploy key)
      read file bytes
      SOPS decrypt in memory (age identities injected; no env vars)
      extract "value" field
      re-encrypt with per-secret DEK under master key
      store.PutSecret
      notify watch bus (wakes any open WatchSecret streams)
  → For each deleted file: store.DeleteSecret + notify
  → store.UpdateSyncState(sha, timestamp)
  → rm -rf temp dir
```

**Periodic reconciliation:**

The `Reconciler` performs a `FullSync` of every registered repository at a configurable interval (default 5 minutes). This catches events missed during downtime (restarts, network partitions) and provides a convergence guarantee independent of webhook delivery.

The reconciler **only runs while the server is unsealed** — it cannot decrypt the stored age keys when sealed. It starts immediately after a successful unseal and is cancelled when the server is sealed.

### Sealed-State Behaviour

- Webhook requests while sealed return HTTP 503 with a `Retry-After` header. GitHub will retry delivery automatically.
- The reconciler goroutine is cancelled on seal and restarted on unseal, so no sync work is attempted against a locked key store.

### `signet bundle push` — local repository upload

**Decision: stream a tar.gz of committed files from CLI to server; server decrypts SOPS on the server side.**

For environments where the secrets repository cannot or should not be pushed to a remote before signet is set up (chicken-and-egg bootstrap), `signet bundle push` packages the local git repository and uploads it directly to signet via a new client-streaming gRPC RPC.

**Trust model**: only SOPS ciphertext leaves the operator's machine. The age private key lives solely in signet's encrypted store, so the server is the only place plaintext is ever visible. This upholds the same guarantee as the webhook-driven flow.

**Why git tree, not filesystem walk**: the CLI reads committed files from the HEAD commit tree (via go-git), not from the working directory. This guarantees only committed secrets are uploaded — no staged, untracked, or stale files can leak.

**Flow:**

```
signet bundle push [repo-path] --secrets-path secrets/

  1. PlainOpen(repo-path) — requires valid git repo with at least one commit
  2. Resolve HEAD → commit → tree
  3. For each .yaml file under secrets_path in the commit tree:
       read file bytes from git object store
       add to in-memory tar.gz
  4. Open SyncBundle gRPC stream
  5. Send SyncBundleChunk{Header{secrets_path, head_sha}}
  6. Send SyncBundleChunk{Data{<chunk>}} in 64 KiB increments
  7. CloseAndRecv() → print SyncBundleResponse summary

Server side (GitOpsServer.SyncBundle):

  1. Authenticate via bearer token (requireToken)
  2. Receive header chunk → extract secrets_path, head_sha
  3. Accumulate data chunks into buffer
  4. extractTarGz to temp dir (path traversal protection; 256 MiB cap)
  5. syncer.SyncFromDir(tmpDir, secretsPath, headSHA)
     → loadIdentities (decrypt stored age keys)
     → walk .yaml files → SOPS decrypt in memory → store.PutSecret
  6. SendAndClose(SyncBundleResponse{added, updated, deleted, sync_sha})
  7. defer os.RemoveAll(tmpDir)
```

`SyncFromDir` is the extracted core of `FullSync` — both code paths share the same walk + SOPS + store logic without duplication.

**Archive security constraints** (`extractTarGz`):
- Rejects paths containing `..` after `filepath.Clean`
- Rejects paths whose resolved dest does not have the extraction dir as a prefix
- Skips symlinks, hard links, devices, and all non-regular-file entries
- Accumulates decompressed byte count; aborts if > 256 MiB

---

## Architecture Summary

```
┌──────────────────────────────────────────────────────────────────┐
│                       Kubernetes Cluster                          │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │           Secret Store Namespace (isolated)               │  │
│  │                                                           │  │
│  │  Signet API Server                                        │  │
│  │    :8443  gRPC + mTLS  (workload secrets)                 │  │
│  │    :8444  gRPC plain   (admin, port-forward only)         │  │
│  │    :8445  HTTP         (GitHub webhooks)                  │  │
│  │                                                           │  │
│  │    - SPIFFE ID extraction + policy evaluation             │  │
│  │    - Envelope decryption (DEK → plaintext, in memory)     │  │
│  │    - SOPS age decryption (in memory, no disk writes)      │  │
│  │    - Periodic git reconciliation (unsealed only)          │  │
│  │    - Audit log writes (HMAC-chained)                      │  │
│  │              │                                            │  │
│  │              ▼  (NetworkPolicy: internal only)            │  │
│  │  CockroachDB StatefulSet                                  │  │
│  │    - AES-256-GCM encryption at rest                       │  │
│  │    - Local PVs; Raft replication                          │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────┐   ┌─────────────┐                             │
│  │ SPIRE Server │   │  Step CA    │                             │
│  │ (kubernetes  │   │  + cert-mgr │                             │
│  │  node +      │   │             │                             │
│  │  workload    │   └─────────────┘                             │
│  │  attestation)│                                               │
│  └──────────────┘                                               │
└──────────────────────────────────────────────────────────────────┘
          │                    │                    │
          ▼                    ▼                    ▼
  ┌──────────────┐   ┌──────────────────┐   ┌────────────────────┐
  │ Operator CLI │   │ Audit Log Server │   │  Git Repository    │
  │ (port-fwd,   │   │ (out-of-cluster, │   │  (SOPS-encrypted   │
  │  unseal +    │   │  HMAC-chained)   │   │   secrets; push    │
  │  repo/key    │   └──────────────────┘   │   triggers sync)   │
  │  management) │                          └────────────────────┘
  └──────────────┘
```

---

## Decision Matrix

| Concern | Decision | Rationale |
|---|---|---|
| Storage backend | CockroachDB | SQL schema, native encryption, mature k8s operator |
| Replication | CockroachDB Raft | HA without separate implementation |
| Network isolation | NetworkPolicy + RBAC | Storage layer invisible outside namespace |
| Client authentication | mTLS + SPIFFE SVIDs | Platform-rooted identity, no long-lived credentials |
| Authorization | Exact-match convention + SPIFFE ID → policy table | No-config access for own namespace/service; glob patterns for all other cases |
| Encryption at rest | Envelope encryption (DEK/KEK/Master) | Per-secret isolation, independent rotation |
| Root of trust | Shamir (distributed), Direct key (simple), Kubernetes Secret (cluster-native), TPM (opt-in) | No hardware dependency; operator choice by threat model |
| Unsealing | Operator CLI via port-forward; independent SA token per keyholder | Auditable, no shared credentials |
| Cluster-native bootstrap | `signet init` stores key in Kubernetes Secret; auto-unseal on restart optional | Single-operator convenience; trust boundary = cluster-admin |
| Persistent storage | Local PVs | No network exposure; application-layer encryption is primary |
| Node identity | SPIRE k8s attestor (baseline); TPM attestor optional | No hardware dependency |
| PKI / CA | Step CA + cert-manager | Short-lived certs, automated rotation |
| Bootstrap ordering | SPIRE before signet | Eliminates TLS bootstrapping dependency |
| Audit logging | HMAC-chained, out-of-cluster | Tamper-evident, survives cluster compromise |
| Secret ingestion | Git + SOPS (age) | Full git history, PR review; no plaintext in repo |
| SOPS key management | age X25519; keypair generated and stored by signet | No external KMS; works air-gapped |
| Sync trigger | GitHub webhook (push) + periodic reconciler | Webhook for immediacy; reconciler for convergence |
| Reconciler lifecycle | Runs only while unsealed | Cannot decrypt age keys when sealed; avoids wasted work |
