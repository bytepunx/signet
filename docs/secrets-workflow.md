# Working with SOPS-encrypted secrets

This document walks through the end-to-end workflow for encrypting a secret
with SOPS and making it available to a workload through signet.

## Overview

```
operator                signet                  workload
────────                ──────                  ────────
generate age key  ──→  stored (encrypted)
write .sops.yaml
encrypt secret
push to git       ──→  webhook / reconciler
                        clone → decrypt → store
                                            ←── GetSecret (mTLS/SVID)
                                                returns plaintext value
```

signet never writes plaintext to disk. Decryption happens in memory; only
re-encrypted ciphertext (AES-256-GCM with a per-secret DEK) is stored in the
database.

---

## Prerequisites

- `sops` ≥ 3.8 installed ([releases](https://github.com/getsops/sops/releases))
- signet running and unsealed
- `signet` CLI configured (`signet config set server http://localhost:8444`)
- A git repository you control (this will hold your encrypted secrets)
- An SSH deploy key with **read-only** access to the repository

---

## Part 1 — Initial setup

### 1.1 Generate the age key in signet

```bash
TOKEN=$(kubectl create token signet-admin -n signet --duration=1h)
signet sops-key get --token "$TOKEN"
```

Output:
```
Public key:  age1abc123def456...
Fingerprint: a1b2c3d4e5f6a1b2
Environment: prod
Created at:  2026-06-24T09:00:00Z
```

When `SIGNET_ENVIRONMENT` is not set the server uses a global key scoped to no
environment; the output will show `(global — add SIGNET_ENVIRONMENT to scope this key to an environment)`.

The private key is stored encrypted inside signet and never exposed. The public
key is the only value you need.

### 1.2 Create or update `.sops.yaml` in your secrets repository

Use `signet sops-key update-config` to generate the correct creation rule for
this environment automatically:

```bash
signet sops-key update-config \
  --file .sops.yaml \
  --secrets-path secrets/ \
  --token "$TOKEN"
```

Output:
```
Updated .sops.yaml
  environment: prod
  path_regex:  ^secrets/prod/
  age key:     age1abc123def456...
```

The resulting `.sops.yaml` contains:

```yaml
# section read by signet tooling; ignored by the sops CLI
environments:
  prod: age1abc123def456...

# section read by the sops CLI for encryption
creation_rules:
  - signet_environment: prod
    path_regex: ^secrets/prod/
    age: age1abc123def456...
```

Run the same command against each environment's signet instance to build a
complete multi-environment file — each invocation adds or updates only its own
environment's entry and leaves all other rules unchanged.

Commit the file:

```bash
git add .sops.yaml && git commit -m "add signet sops config"
```

### 1.3 Generate an SSH deploy key

```bash
ssh-keygen -t ed25519 -C "signet-deploy" -f signet_deploy_key -N ""
```

Add the **public key** to your repository:
- GitHub → Repository → Settings → Deploy keys → Add deploy key
- Leave "Allow write access" unchecked

### 1.4 Register the repository with signet

Pass `--setup-webhook` to have signet create the GitHub webhook automatically.
signet tries `gh` CLI first, then falls back to a GitHub token, logging each
step so you always know what happened.

**Recommended — automatic webhook setup:**

```bash
# Option A: gh CLI (no token required — gh handles authentication)
signet repo add \
  --name         my-secrets \
  --repo-url     git@github.com:myorg/my-secrets \
  --branch       main \
  --secrets-path secrets/ \
  --deploy-key   ./signet_deploy_key \
  --setup-webhook \
  --token        "$TOKEN"

# Option B: explicit GitHub token (classic PAT, admin:repo_hook scope)
signet repo add \
  --name         my-secrets \
  --repo-url     git@github.com:myorg/my-secrets \
  --deploy-key   ./signet_deploy_key \
  --setup-webhook \
  --github-token "$GITHUB_TOKEN" \
  --token        "$TOKEN"
```

Example output with `gh` CLI:
```
Repository registered.
  ID:             550e8400-e29b-41d4-a716-446655440000
  Webhook URL:    https://signet.example.com/webhook/github/550e8400-...
  Webhook secret: a3f1b2c4d5e6...

This secret will not be shown again.

Setting up webhook automatically for myorg/my-secrets...
  Checking for gh CLI... found (/usr/bin/gh).
  Calling `gh api repos/myorg/my-secrets/hooks`...
  Webhook created.
```

Example output when `gh` is absent and `GITHUB_TOKEN` is set:
```
Setting up webhook automatically for myorg/my-secrets...
  Checking for gh CLI... not found.
  gh CLI not available, GITHUB_TOKEN is set in environment, calling GitHub's REST API...
  Webhook created.
```

**Manual webhook setup (without `--setup-webhook`):**

```bash
signet repo add \
  --name         my-secrets \
  --repo-url     git@github.com:myorg/my-secrets \
  --deploy-key   ./signet_deploy_key \
  --token        "$TOKEN"
```

signet prints the webhook URL and secret, then you configure it in GitHub:

1. Open your repository → **Settings → Webhooks → Add webhook**
2. **Payload URL:** the URL printed by `signet repo add`
3. **Content type:** `application/json`
4. **Secret:** the secret printed by `signet repo add`
5. **Events:** Just the push event
6. **Active:** checked → **Add webhook**

Save the webhook secret. Delete the deploy key from disk:
```bash
rm signet_deploy_key
```

---

## Part 2 — Creating and encrypting secrets

### Environment separation

When running multiple clusters (e.g. prod, staging, dev), each signetd instance
is given a distinct environment label:

```yaml
# signetd Deployment env
- name: SIGNET_ENVIRONMENT
  value: prod    # or: staging, dev, etc.
```

Each environment has its **own age keypair**. After setting `SIGNET_ENVIRONMENT`,
rotate the key to generate an environment-scoped key:

```bash
TOKEN=$(kubectl create token signet-admin -n signet --duration=1h)
signet sops-key rotate --token "$TOKEN"
# New public key:  age1xyz789...
# Environment:     prod
```

Use **separate subdirectories in the same repository** for each environment:

```
secrets/
├── prod/
│   └── payments/api/stripe-key.yaml
├── staging/
│   └── payments/api/stripe-key.yaml
└── dev/
    └── payments/api/stripe-key.yaml
```

Use `signet sops-key update-config` against **each** environment's signet
instance to build the `.sops.yaml` incrementally:

```bash
# Against the prod signet instance
SIGNET_SERVER=prod.signet:8444 signet sops-key update-config --token "$PROD_TOKEN"

# Against the staging instance
SIGNET_SERVER=staging.signet:8444 signet sops-key update-config --token "$STAGING_TOKEN"

# Against the dev instance
SIGNET_SERVER=dev.signet:8444 signet sops-key update-config --token "$DEV_TOKEN"
```

Each invocation adds its environment's entry and leaves all others untouched.
The resulting `.sops.yaml` has an `environments` map (signet-specific, ignored
by the `sops` CLI) plus one creation rule per environment:

```yaml
# section read by signet tooling
environments:
  prod:    age1prod111...
  staging: age1staging222...
  dev:     age1dev333...

# section read by the sops CLI for encryption
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

Each signet instance decrypts **only the files encrypted with its environment's
key**. A production signet server cannot decrypt staging or dev secrets even if
it has access to the repository.

Configure each instance's `--secrets-path` to point to its subdirectory:

```bash
signet repo add \
  --name         my-secrets-prod \
  --repo-url     git@github.com:myorg/my-secrets \
  --secrets-path secrets/prod/ \
  --deploy-key   ./signet_deploy_key \
  --setup-webhook \
  --token        "$TOKEN"
```

> **Single-environment setups:** Leave `SIGNET_ENVIRONMENT` unset. signet uses a
> global (unscoped) key. All existing workflows apply without any path changes.

---

### File path convention

The path within the repository determines how signet identifies the secret.
With environment separation the path includes the environment prefix:

```
secrets/<env>/<namespace>/<service>/<name>.yaml
```

Without environment separation (`SIGNET_ENVIRONMENT` unset):

```
secrets/<namespace>/<service>/<name>.yaml
```

Examples (environment-separated):
```
secrets/prod/payments/api/stripe-key.yaml    → namespace=payments, service=api, name=stripe-key
secrets/prod/infra/redis/password.yaml       → namespace=infra,    service=redis, name=password
secrets/staging/payments/api/stripe-key.yaml → namespace=payments, service=api, name=stripe-key
```

Examples (single-environment):
```
secrets/payments/api/stripe-key.yaml   → namespace=payments, service=api, name=stripe-key
secrets/infra/redis/password.yaml      → namespace=infra,    service=redis, name=password
secrets/platform/auth/signing-key.yaml → namespace=platform, service=auth,  name=signing-key
```

### Secret file format

Each file must have a single top-level `value` key:

```yaml
value: the-actual-secret-value
```

### Encrypt a new secret

```bash
# Create directory structure
mkdir -p secrets/payments/api

# Write the plaintext file
cat > secrets/payments/api/stripe-key.yaml <<'EOF'
value: sk_live_AbCdEfGh1234567890
EOF

# Encrypt in place (reads .sops.yaml to find the key)
sops --encrypt --in-place secrets/payments/api/stripe-key.yaml

# Verify it is encrypted (value should now be ENC[...])
head secrets/payments/api/stripe-key.yaml

# Commit and push
git add secrets/payments/api/stripe-key.yaml
git commit -m "add stripe key for payments/api"
git push
```

After the push, GitHub fires a webhook event. signet clones the repository at
the pushed SHA, decrypts the changed file, and stores the secret.

### Worked example

A realistic set of secrets for a payments service:

```
secrets/
└── payments/
    └── api/
        ├── stripe-key.yaml          # Stripe API key
        ├── db-password.yaml         # CockroachDB password
        └── jwt-signing-key.yaml     # JWT private key (PEM-encoded)
```

Each file:
```yaml
# stripe-key.yaml (after encryption)
stripe-key: ENC[AES256_GCM,data:abc...,tag:xyz...,type:str]
sops:
  kms: []
  age:
    - recipient: age1abc123...
      enc: |
        -----BEGIN AGE ENCRYPTED FILE-----
        ...
  ...
```

---

## Part 3 — Verifying secret retrieval

### Trigger initial sync

The webhook handles incremental updates. For secrets already in the repository
before the webhook was configured, trigger a full sync:

```bash
REPO_ID=550e8400-e29b-41d4-a716-446655440000
signet repo sync --id "$REPO_ID" --token "$TOKEN"
```

### Check from a workload

From a pod with the correct SPIFFE ID, the secret is available via the gRPC
API. If the workload's SPIFFE ID encodes the same namespace and service account
as the secret's namespace/service, no explicit policy is required (see
[policies.md](policies.md) for cross-service access patterns):

```go
// Go client using go-spiffe for mTLS
source, _ := workloadapi.NewX509Source(ctx)
defer source.Close()

creds := grpccredentials.MTLSClientCredentials(source, source,
    tlsconfig.AuthorizeID(spiffeid.RequireIDFromString("spiffe://cluster.local/signet")))

conn, _ := grpc.Dial("signet.signet.svc.cluster.local:8443", grpc.WithTransportCredentials(creds))
client := signetv1.NewSecretsServiceClient(conn)

resp, _ := client.GetSecret(ctx, &signetv1.GetSecretRequest{
    Namespace: "payments",
    Service:   "api",
    Name:      "stripe-key",
})
// resp.Value is the plaintext secret
```

---

## Part 4 — Key rotation

Rotate the age key periodically or after a suspected compromise.

### 4.1 Rotate the key

```bash
signet sops-key rotate --token "$TOKEN"
```

Output:
```
New public key:  age1xyz789...
New fingerprint: f6e5d4c3b2a1f6e5
Environment:     prod
Old key retained for decryption: age1abc123...
Re-encrypt your SOPS files with the new key, then run 'signet sops-key prune'.
```

### 4.2 Update `.sops.yaml`

```bash
# Replace the old key with the new one
sed -i 's/age: age1abc123.*/age: age1xyz789.../' .sops.yaml
```

### 4.3 Re-encrypt all secret files

```bash
find secrets/ -name '*.yaml' | while read f; do
  sops --rotate --in-place "$f"
done

git add -A
git commit -m "rotate sops key to $(signet sops-key get --token "$TOKEN" | awk '/Fingerprint:/{print $2}')"
git push
```

signet will re-sync the re-encrypted files using the new key. The old key
continues to decrypt existing files until pruned.

### 4.4 Prune the old key

After confirming the sync is complete and all secrets are accessible:

```bash
# Find the inactive key's public key
signet sops-key list --token "$TOKEN"

# Prune it
signet sops-key prune \
  --public-key age1abc123def456... \
  --token "$TOKEN"
```

---

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| Secret not appearing after push | Check webhook delivery log in GitHub; check `signet repo list` for last-sync SHA; verify path convention |
| Sync error: `invalid path` | File is not under the expected `secrets/[env/]<ns>/<svc>/<name>.yaml` structure |
| Sync error: `decryption failed` | The SOPS file is encrypted to a key signet does not hold; check `.sops.yaml` matches the public key for this environment |
| Sync error: `no age keys configured for environment "prod"` | `SIGNET_ENVIRONMENT=prod` is set but no key has been generated; run `signet sops-key rotate` against this instance |
| `GetSecret` returns `NOT_FOUND` | Secret not yet synced, or namespace/service/name mismatch |
| `GetSecret` returns `PERMISSION_DENIED` | Workload SPIFFE ID does not match the secret's namespace/service exactly, and no explicit policy grants access; see [policies.md](policies.md) |
| Server sealed | Unseal signet; GitHub will retry the webhook automatically within 30 seconds |
| Wrong environment decrypting secrets | `SIGNET_ENVIRONMENT` mismatch between the signet instance and the `.sops.yaml` path rule; confirm `signet sops-key get` shows the expected environment |
