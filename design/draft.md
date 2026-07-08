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
secrets (namespace, service, secret_name, version, encrypted_dek, ciphertext, expires_at, metadata, kek_id)
access_policies (spiffe_id, namespace, secret_pattern, permissions)
audit_log (ts, spiffe_id, action, namespace, secret_name, outcome, peer_ip)  -- TTL: 90 days
key_encryption_keys (id, wrapped_kek, is_active, created_at, deactivated_at)
key_check_value (id, ciphertext, created_at)  -- singleton row
```

`secrets.kek_id` is nullable: rows written before the KEK tier was introduced have their DEK
wrapped directly under the master key (`kek_id IS NULL`); decryption handles both forms (see
Section 5).

**Expiry enforcement:** reads (`GetSecret`, `GetServiceBundle`) filter on
`expires_at IS NULL OR expires_at > now()`. If a secret's newest version has expired but an
older version has not, the newest non-expired version is served rather than treating the
secret as absent; if every version has expired the secret is treated as not found. As of this
writing, no ingestion path (git/SOPS sync, `signet bundle push`) sets `expires_at` — it is
enforced at read time but not yet populated by any write path; a future admin/CLI surface for
setting per-secret expiry is expected to build on this enforcement.

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
exactly matches the secret's `namespace` and `service` fields, AND whose SPIFFE ID's trust domain
matches this signet instance's configured trust domain, is permitted automatically — no row
in `access_policies` is required. This covers the primary usage pattern without any operator
configuration. Explicit policies are only necessary for cross-service or cross-namespace access.
(The trust domain check here is belt-and-suspenders with the TLS-layer check SPIRE credentials
already perform — it keeps the authorization decision self-contained.)

Policies support prefix/glob matching against a three-segment
`namespace/service/secret_name` target, enabling coarse or fine-grained control. The service
segment may itself be a glob (`*`) to grant across every service in a namespace:

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

### Associated Data Binding (AAD)

Every AES-256-GCM operation in the envelope hierarchy — secret ciphertext, DEK wrap, KEK
wrap, and the key-check value (Section 6) — is bound via GCM additional authenticated data
(AAD) to the logical identity of what it protects: `(namespace, service, secret_name)` for
secret ciphertext and DEK wraps, a fixed context tag for KEK wraps and the key-check value,
and `(repository name)` / `(SOPS public key)` for the webhook secret, deploy key, and SOPS
private keys respectively. AAD is not stored — it is recomputed from the row being read and
supplied to `Decrypt`/`UnwrapKey`, so a blob copied from one row into another (by a party with
database write access but no key material) fails GCM authentication instead of silently
decrypting under the destination's identity. This closes a swap/substitution class of attack
that key material alone does not prevent.

Data encrypted before AAD binding was introduced (empty `kek_id`, or any artifact from an
older signet version) has no AAD to check against. Decryption of such data falls back to the
legacy nil-AAD form and logs a warning identifying the artifact so operators can identify what
still needs to be re-synced (secrets, via a full `signet gitops` sync) or re-registered
(repository webhook secret / deploy key, SOPS keys) to gain the new binding. New writes always
use AAD; there is no way to write new data in the legacy unbound form.

### KEK and Master Key Rotation

```bash
# Re-wrap every secret's DEK under a newly generated KEK. Cheap: touches only
# DEKs (small), never re-encrypts secret ciphertext blobs.
signet kek rotate
signet kek list
signet kek prune --id <old-kek-id>   # once no secret still references it

# Re-wrap every KEK (and the key-check value) under a new master key, then
# adopt it. Even cheaper: touches only the (typically small) set of KEKs, never
# secrets or DEKs directly. The operator supplies the new key (or lets the CLI
# generate one, shown once) and is responsible for redistributing it to Shamir
# keyholders or updating the Kubernetes auto-unseal Secret afterward.
signet master-key rotate --new-key-file ./new-master.key
```

`RotateKEK` deactivates (does not delete) the previous KEK so any DEK not yet re-wrapped
remains decryptable; `PruneKEK` refuses to delete the active KEK or one still referenced by
any secret. `RotateMasterKey` re-wraps the database's KEK and key-check rows inside one
transaction before adopting the new key in memory; if adopting the new key in memory fails
after that transaction commits, signetd makes a best-effort attempt to roll the database back
to the previous wraps so the still-loaded old key remains authoritative.

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

**etcd encryption-at-rest is a prerequisite, not optional.** The master key is stored in the
Secret's `master.key` field as base64 — not encrypted — and Kubernetes Secrets are themselves
stored in etcd unencrypted unless the cluster explicitly configures an `EncryptionConfiguration`
(https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/). Without that, the master
key is recoverable by anyone with etcd access (backups included), independent of Kubernetes
RBAC entirely. signetd logs a warning at startup when auto-unseal is enabled, since it cannot
verify etcd's encryption configuration itself. Exclude the `master.key` Secret from any backup
or GitOps export tooling.

#### Bootstrap: `signet init`

A single CLI command handles the entire cluster preparation and unseal workflow:

```
signet init [flags]
  --namespace      Kubernetes namespace where signet is deployed (default: signet)
  --key-secret     Name of the Kubernetes Secret to create/read (default: signet-master-key)
  --kube-context   kubectl context to use (default: current context)
  --server         signet admin endpoint (default: from config)
  --token          admin Bearer token
  --force          Regenerate key and overwrite existing Secret (DESTRUCTIVE — see below)
  --yes            Skip the interactive confirmation prompt required by --force
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

**`--force` is destructive and irreversible:** regenerating the key for an *existing* Secret
orphans every secret currently encrypted under the old key — there is no way to recover them
afterward. `signet init --force` therefore requires interactive confirmation (type `yes`)
unless `--yes` is passed for scripted/CI use. This confirmation is skipped when `--force` is
combined with `--dry-run`, and does not apply when no Secret exists yet (creating a Secret for
the first time is not destructive).

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

### Key-Check Value

Every unseal path (direct key, Shamir reconstruction, and Kubernetes auto-unseal) verifies the
candidate master key against a stored **key-check value** before the server is declared
operational: a fixed constant encrypted under the master key with AAD binding, persisted on
first successful unseal. On every later unseal, the candidate key must decrypt this value; on
mismatch the server immediately re-seals and the unseal call returns an error, rather than
running with a key that decrypts nothing. This distinguishes "wrong key supplied" (immediate,
clear failure) from "data corruption" (which would otherwise only surface once a workload
first tries to read a secret) and prevents a stale Kubernetes Secret or an incorrect Shamir
share set from silently leaving the server in a half-functional state.

### Mechanism 4: TPM / vTPM Auto-Unseal (opt-in) — NOT YET IMPLEMENTED

**Status: design only.** `UnsealWithTPM` (`internal/unseal/tpm.go`, built with `-tags tpm`)
currently returns "not yet implemented"; the default (non-`tpm`-tagged) build returns
`ErrTPMNotSupported` unconditionally. Nothing below this line is available to operators yet —
it describes the intended design once implemented, not current behavior. Do not plan a
deployment around TPM auto-unseal until this note is removed.

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

### Admin Authorization

A valid, authenticated Kubernetes ServiceAccount token is necessary but not sufficient to call
any admin RPC (`AdminService` or `GitOpsService`). After `TokenReview` confirms the token is
authentic, signetd authorizes the resulting identity via either of two complementary
mechanisms — either is sufficient:

1. **Allowlist** (`SIGNET_ADMIN_SUBJECTS`) — a comma-separated list of
   `serviceaccount:<namespace>:<name>` or `group:<name>` entries, checked first as a fast
   path requiring no extra API call.
2. **SubjectAccessReview** — if not on the allowlist, signetd asks the cluster whether the
   caller's identity has been granted verb `administer` on the synthetic resource
   `adminoperations` in API group `signet.io` (no CRD required — RBAC evaluates the tuple
   regardless of whether a real API resource backs it):

   ```yaml
   apiVersion: rbac.authorization.k8s.io/v1
   kind: ClusterRole
   metadata:
     name: signet-admin-operator
   rules:
     - apiGroups: ["signet.io"]
       resources: ["adminoperations"]
       verbs: ["administer"]
   ```

If neither mechanism grants access, the RPC fails with `PermissionDenied` — the request never
reaches unseal, seal, KEK/master-key rotation, or GitOps logic. `SIGNET_KUBE_AUDIENCES` must
be non-empty (default `signet`); an empty audience list would make `TokenReview` accept a
token bearing any audience, so signetd refuses to start rather than silently widen who can
even attempt admin authentication.

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

**Transport security:** the `signet` CLI sends the bearer token in cleartext only to a loopback
`--server` address (the `kubectl port-forward` case above, where the tunnel itself is already
protected by kubeconfig credentials). Any other address is automatically upgraded to TLS —
using the system trust store, or a CA supplied via `--ca` — before the token is ever sent; if
the server's certificate cannot be verified, the connection fails rather than silently falling
back to plaintext. `--tls` forces TLS even for a loopback address.

### Local Development

For local clusters (kind, minikube, k3s), the admin endpoint may be configured to accept
connections on loopback without TLS using `--dev-mode`. This mode refuses to start if the
bind address is not loopback and must never be used in production deployments.

---

## 11. Audit Logging

**Decision: Structured append-only logs forwarded out-of-cluster in near-real-time, with
HMAC chaining for tamper detection.**

- Every secret access (permitted or denied) written to `audit_log` table and forwarded — this
  includes `GetSecret`/`WatchSecret` as well as `GetConfig`/`GetServiceConfig`/
  `WatchServiceConfig`/`GetServiceBundle`/`WatchServiceBundle`; the two bundle actions use the
  sentinel secret names `<config>`/`<bundle>` since they span an entire document rather than
  one named secret. `GetServiceBundle` in particular decrypts every secret for a service in one
  call, so its audit entry is written before the bundle is returned to the caller.
- HMAC chaining: each log entry includes an HMAC of the previous entry — retroactive tampering
  is detectable
- **Fail-closed by default** (`SIGNET_AUDIT_FAIL_CLOSED`, default `true`): if the audit write
  for an otherwise-permitted access fails, the access is denied (`codes.Unavailable`) rather
  than served without a durable audit trail. A denied access is unaffected — its own audit
  write failing does not change the fact that it was already being denied. Operators who accept
  the availability tradeoff may set this to `false` to fail open instead.
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

### Plain Config vs. Secrets

Files under a repository's `secrets_path` are SOPS-encrypted and, once synced, receive the
full envelope-encryption treatment described in Section 5 (per-secret DEK, KEK wrap, AAD
binding). Files under the separate, optional `config_path` are plain YAML — **not** SOPS-
encrypted in the repository and **not** enveloped at rest — they are stored as plaintext JSON,
protected only by CockroachDB's storage-layer encryption. This is intentional: config exists
for non-sensitive service configuration (feature flags, timeouts, endpoints) that benefits from
being human-readable and diffable in git without a decryption step. Operators must not place
secret material in `config_path` files; `secrets_path` is the only ingestion route with
per-secret envelope encryption.

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

**No-op dedup:** before re-encrypting and storing a secret, `storeSecret` compares the freshly
SOPS-decrypted plaintext against the currently stored version. If it is identical *and* the
stored row is already wrapped under the current active KEK (which implies it already has AAD
binding — the two ship together), the write is skipped entirely, bounding the otherwise-
unbounded version growth from re-syncing unchanged secrets every reconciliation pass. A row on
a rotated-away KEK or predating the KEK tier (`kek_id` empty) is never treated as unchanged, so
it is naturally rewritten onto the current epoch the next time it is synced — this is how the
AAD/KEK migration (Section 5) converges without a separate migration job.

### Sealed-State Behaviour

- Webhook requests while sealed return HTTP 503 with a `Retry-After` header. GitHub will retry delivery automatically.
- The reconciler goroutine is cancelled on seal and restarted on unseal, so no sync work is attempted against a locked key store.

### Webhook Hardening

The webhook endpoint is necessarily unauthenticated (GitHub does not present a bearer token,
only an HMAC signature over the body) and reachable by anyone who can route to it, so it is
rate-limited (20 req/s sustained, burst 40, globally per signetd process) before any other work
— including the master-key decrypt needed to verify the signature — so a caller cannot force
unbounded crypto/CPU work per second. An unknown `repo_id` and a known repo with a bad HMAC
signature return the identical `401` response; the distinguishing detail is only logged
server-side, so the endpoint cannot be used to enumerate valid repository IDs by response alone.

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
| Encryption at rest | Envelope encryption (DEK/KEK/Master), AAD-bound to logical identity | Per-secret isolation, independent rotation, blocks cross-row ciphertext substitution |
| Root of trust | Shamir (distributed), Direct key (simple), Kubernetes Secret (cluster-native), TPM (opt-in) | No hardware dependency; operator choice by threat model |
| Unsealing | Operator CLI via port-forward; independent SA token per keyholder; key-check value verified on every unseal | Auditable, no shared credentials; wrong-key unseals fail fast and re-seal |
| Admin authorization | TokenReview (authenticate) + allowlist or SubjectAccessReview (authorize) | Valid token alone is not sufficient for admin RPCs; delegable to cluster RBAC |
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
