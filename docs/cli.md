# signet CLI reference

The `signet` binary is the operator interface to a running signetd instance.
All commands communicate with the admin gRPC endpoint (default
`http://localhost:8444`), which is only reachable via `kubectl port-forward`.

## Global flags

| Flag | Default | Description |
|---|---|---|
| `--server` | value from `signet config` | Admin endpoint URL |
| `--token` | `""` | Bearer token for admin auth; alternative to `--token-file` |
| `--token-file` | `""` | File containing the Bearer token |
| `--ca` | `""` | Path to a PEM CA certificate to trust for the admin server; implies TLS |
| `--tls` | `false` | Use TLS even when connecting to a loopback address |

When neither `--token` nor `--token-file` is supplied, signet attempts to use
a token from the environment variable `SIGNET_TOKEN`.

### Transport security

The bearer token is only sent in cleartext to a **loopback** `--server`
address (`localhost`, `127.0.0.1`, `::1`) — the documented `kubectl
port-forward` workflow, where the tunnel itself is already protected by
kubeconfig credentials. Any other address is automatically upgraded to TLS,
using the system trust store or the CA supplied via `--ca`. If the server's
certificate can't be verified, the connection fails outright rather than
silently falling back to plaintext. Use `--tls` to force TLS for a loopback
address too (e.g. to test a TLS-terminating proxy locally).

---

## `signet init`

Bootstrap a cluster for single-key operation. `signet init` generates a 32-byte
master key, stores it in a Kubernetes Secret, and unseals the server in one
step. It is the recommended approach for dev clusters and single-operator
setups.

> **Security note:** the master key is stored in a Kubernetes Secret. Any
> principal with cluster-admin access can read it. For production
> multi-operator environments, use Shamir unseal instead.

```
Usage: signet init [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--namespace` / `-n` | `signet` | Kubernetes namespace where signet is deployed |
| `--key-secret` | `signet-master-key` | Name of the Kubernetes Secret to create or reuse |
| `--kube-context` | `""` | kubeconfig context to use; defaults to current context |
| `--force` | `false` | Regenerate the master key and overwrite the existing Secret. **Destructive** — see [`--force` semantics](#--force-semantics) below |
| `--dry-run` | `false` | Print what would happen without creating any resources or contacting the server |
| `--yes` | `false` | Skip the interactive confirmation prompt required by `--force` |

### What it does

1. Checks the server's current seal state; exits immediately with `signet is
   already unsealed.` if it's already unsealed (even with `--force`).
2. Looks for an existing master key in `--key-secret` / `--namespace`.
3. If not found, generates 32 random bytes as the master key and creates the
   Secret. If found and `--force` is not set, reads the existing key as-is.
   If found and `--force` **is** set, regenerates the key and overwrites the
   Secret (see below).
4. Sends the resulting key to signetd via `UnsealKey`.
5. Re-checks status and prints the key source (`created` / `regenerated` /
   `existing`) and resulting server state.

### Example — fresh cluster (Secret not found)

```
$ signet init --server localhost:8444 --token "$TOKEN"
signet state: sealed
Secret "signet-master-key" not found in namespace "signet" — generating new 32-byte master key.
Created Secret signet/signet-master-key.
WARNING: Secret signet/signet-master-key contains the plaintext master key.
         Restrict access with Kubernetes RBAC. For production, consider
         encrypting this Secret with Sealed Secrets or External Secrets.
signet unsealed using created key from Secret signet/signet-master-key. State: unsealed
```

### Example — re-running when already unsealed

```
$ signet init --server localhost:8444 --token "$TOKEN"
signet is already unsealed.
```

### `--force` semantics

`--force` **regenerates** the master key and overwrites the existing Secret —
it does not reuse the current key. This is destructive: every secret and KEK
currently wrapped under the old master key becomes permanently undecryptable
once the new key replaces it in the Secret. Because of that, `--force` always
prints a warning and requires typing `yes` at an interactive prompt, unless
`--yes` is passed to skip it (for scripted/CI use, once you're certain).

`--force` is for replacing a *lost or compromised* key, not for routine
re-unsealing after a pod restart — for that, just run `signet init` again with
no flags; it reuses the existing Secret unchanged. To actually rotate the
master key on a running server without losing access to existing data
(recommended over `--force`), use [`signet master-key rotate`](#signet-master-key-rotate)
instead, which re-wraps everything in place before the key changes.

### `--dry-run` for CI validation

`--dry-run` performs all checks (context resolution, namespace lookup, server
reachability) and prints every action it would take, without writing any
Kubernetes resources or calling the unseal RPC. Use this in CI to verify the
command would succeed without side effects.

```bash
$ signet init --dry-run
signet state: sealed
Secret "signet-master-key" not found in namespace "signet" — generating new 32-byte master key.
[dry-run] would create Secret signet/signet-master-key with new master key.
[dry-run] would unseal signet with key from Secret signet/signet-master-key.
[dry-run] no changes made.
```

---

## `signet config`

Manage local CLI configuration stored in `~/.config/signet/config.yaml`.

### `signet config set <key> <value>`

```
Usage: signet config set <key> <value>
```

Supported keys:

| Key | Description |
|---|---|
| `server` | Admin endpoint URL, e.g. `http://localhost:8444` |

**Example:**
```bash
signet config set server http://localhost:8444
```

---

## `signet status`

Show the server's current seal state.

```
Usage: signet status
```

**Example output (sealed):**
```
state=sealed
```

**Example output (unsealing — Shamir mode):**
```
state=unsealing shares=1/3
```

**Example output (unsealed):**
```
state=unsealed
```

---

## `signet unseal`

Unseal the server. Two sub-commands cover the two unseal modes.

### `signet unseal key` — Direct-key mode

```
Usage: signet unseal key --key-file <path> [--token <token>]
```

Flags:

| Flag | Required | Description |
|---|---|---|
| `--key-file` | Yes | Path to the binary master key file (32 bytes) |
| `--token` | | Admin Bearer token |

**Example:**
```bash
TOKEN=$(kubectl create token signet-admin -n signet --duration=1h)
signet unseal key --key-file master.key --token "$TOKEN"
# Server unsealed.
```

### `signet unseal share` — Shamir mode

```
Usage: signet unseal share {--share <hex> | --share-file <path>} [--token <token>]
```

Flags:

| Flag | Required | Description |
|---|---|---|
| `--share` | One of | Shamir share as a hex string |
| `--share-file` | One of | Path to a file containing a hex-encoded share |
| `--token` | | Admin Bearer token |

Each key holder runs this command independently. The server unseals once the
threshold is reached.

**Example:**
```bash
signet unseal share --share-file /secure/my.share --token "$TOKEN"
# Key accepted (1/3 shares satisfied).
```

---

## `signet seal`

Seal the server — wipes the master key from memory. All active connections are
rejected until the server is unsealed again.

```
Usage: signet seal [--token <token>]
```

**Example:**
```bash
signet seal --token "$TOKEN"
# Server sealed.
```

---

## `signet kek`

Manage the key-encryption-key (KEK) that sits between the master key and each
secret's per-secret data encryption key (DEK): `Master -> KEK -> DEK ->
secret`. Rotating the KEK re-wraps every secret's DEK without touching any
secret's ciphertext, so it's cheap regardless of how many secrets exist.

### `signet kek rotate`

Generate a new KEK, deactivate the current one (retained so any DEK not yet
re-wrapped can still be decrypted), and re-wrap every secret's DEK from the
old KEK to the new one. On a fresh deployment with no active KEK yet, this
simply provisions the first one — there's no separate "kek init" step.

```
Usage: signet kek rotate [--token <token>]
```

**Example — rotating an existing KEK:**
```
$ signet kek rotate --token "$TOKEN"
New KEK:             7c3f8e21-...
Old KEK (retained):  1a2b3c4d-...
Secrets re-wrapped:  482
Once confirmed, run 'signet kek prune 1a2b3c4d-...' to remove the old KEK.
```

**Example — first KEK on a fresh deployment:**
```
$ signet kek rotate --token "$TOKEN"
New KEK:             7c3f8e21-...
```

The old KEK is **not** deleted automatically. Once you've confirmed the
rotation succeeded, run `signet kek prune <old-kek-id>` to remove it.

### `signet kek list`

List all KEKs — active and retained-for-decryption.

```
Usage: signet kek list [--token <token>]
```

**Example output:**
```
[active]   7c3f8e21-...  created=2026-06-24T10:00:00Z  deactivated=-
[inactive] 1a2b3c4d-...  created=2026-01-10T09:00:00Z  deactivated=2026-06-24T10:00:00Z
```

If no KEK exists yet, prints `No KEKs found. Run 'signet kek rotate' to
provision one.`

### `signet kek prune`

Permanently delete an inactive, unreferenced KEK.

```
Usage: signet kek prune --id <id> [--token <token>]
```

Flags:

| Flag | Required | Description |
|---|---|---|
| `--id` | Yes | KEK id to prune (from `signet kek list`) |

Refuses to delete:
- **the active KEK** — `cannot prune the active kek; rotate first`
- **a KEK still referenced by any secret's DEK wrap** —
  `cannot prune kek <id>: still referenced by <n> secret(s)`. Run
  `signet kek rotate` first and confirm `Secrets re-wrapped` accounts for
  every secret before pruning.

**Example:**
```bash
signet kek prune --id 1a2b3c4d-... --token "$TOKEN"
# kek 1a2b3c4d-... pruned
```

---

## `signet master-key`

Manage the top-level master key that wraps every KEK and the key-check value.

### `signet master-key rotate`

Re-wrap every KEK (and the key-check value) under a new master key, then
adopt it as the server's active master key. Secrets and their DEKs are never
touched directly — only their KEK layer — which is why this is cheap
regardless of how many secrets are stored.

```
Usage: signet master-key rotate [--new-key-file <path>] [--yes] [--token <token>]
```

Flags:

| Flag | Default | Description |
|---|---|---|
| `--new-key-file` | `""` | Path to a binary file containing the new 32-byte master key. If omitted, a random key is generated and printed once |
| `--yes` | `false` | Skip the interactive confirmation prompt |

This is a **destructive, high-impact operation on the live server** and
always requires typing `yes` at a prompt unless `--yes` is passed. It does
**not** redistribute the new key to Shamir keyholders or update a Kubernetes
auto-unseal Secret — after rotating, you are responsible for:

- **Shamir mode:** regenerate and redistribute new shares from the new key.
- **Kubernetes auto-unseal:** update the master-key Secret with the new value
  (e.g. `signet init --force`), or auto-unseal will keep trying the old key
  and fail the key check on next restart.
- **Direct-key mode:** securely store the new key in place of the old one.

If adopting the new key in memory fails *after* the database has already been
updated, signetd rolls the database back to the previous wraps so the
still-loaded old key remains authoritative — but this is a narrow,
best-effort safeguard, not a substitute for checking `signet status` after
rotating.

**Example — generating a new key:**
```
$ signet master-key rotate --token "$TOKEN"
WARNING: this rotates the live master key. Every KEK (and the key-check
value) will be re-wrapped under the new key. You are responsible for
redistributing the new key to Shamir keyholders or updating the
Kubernetes auto-unseal Secret afterward — see 'signet master-key rotate --help'.
Type 'yes' to continue: yes
master key rotated; redistribute the new key to keyholders (Shamir) or update the cluster Secret (auto-unseal) as appropriate
KEKs re-wrapped: 3

New master key (shown once — record it securely now):
9f8e7d6c5b4a...
```

**Example — supplying your own key, non-interactively:**
```bash
signet master-key rotate --new-key-file ./new-master.key --yes --token "$TOKEN"
```

---

## `signet sops-key`

Manage the age encryption keypair used for SOPS-encrypted secrets.

### `signet sops-key get`

Print the currently active age public key.

```
Usage: signet sops-key get [--token <token>]
```

**Example output (environment-scoped instance):**
```
Public key:  age1abc123def456...
Fingerprint: a1b2c3d4e5f6a1b2
Environment: prod
Created at:  2026-01-15T09:00:00Z
```

**Example output (global, no SIGNET_ENVIRONMENT set):**
```
Public key:  age1abc123def456...
Fingerprint: a1b2c3d4e5f6a1b2
Environment: (global — add SIGNET_ENVIRONMENT to scope this key to an environment)
Created at:  2026-01-15T09:00:00Z
```

Use the public key value in your `.sops.yaml` creation rules.

### `signet sops-key rotate`

Generate a new age keypair scoped to this instance's environment and deactivate
the current environment-scoped key.

```
Usage: signet sops-key rotate [--token <token>]
```

**Example output:**
```
New public key:  age1xyz789...
New fingerprint: f6e5d4c3b2a1f6e5
Environment:     prod
Old key retained for decryption: age1abc123...
Re-encrypt your SOPS files with the new key, then run 'signet sops-key prune'.
```

The old key is kept for decryption until explicitly pruned. This allows
re-encryption of SOPS files before the old key is removed.

### `signet sops-key list`

List all age keys visible to this instance's environment — both the active key
and any inactive keys retained for decryption.

```
Usage: signet sops-key list [--token <token>]
```

**Example output:**
```
[active]   age1xyz789...  env=prod      fp=f6e5d4c3b2a1f6e5  created=2026-06-01T09:00:00Z  deactivated=-
[inactive] age1abc123...  env=prod      fp=a1b2c3d4e5f6a1b2  created=2026-01-15T09:00:00Z  deactivated=2026-06-01T09:00:00Z
[inactive] age1old000...  env=(global)  fp=deadbeef12345678  created=2025-11-01T14:30:00Z  deactivated=2026-01-15T09:00:00Z
```

### `signet sops-key update-config`

Create or update a `.sops.yaml` file with the active age key for this signet
instance's environment. Run once per environment to build a multi-environment
`.sops.yaml` incrementally; each invocation only touches its own environment's
entries and preserves all other content.

```
Usage: signet sops-key update-config [--file <path>] [--secrets-path <path>] [--print] [--token <token>]
```

Flags:

| Flag | Default | Description |
|---|---|---|
| `--file` | `.sops.yaml` | Path to the `.sops.yaml` file to create or update |
| `--secrets-path` | `secrets/` | Secrets directory prefix within the repository |
| `--print` | false | Print the result to stdout instead of writing to file |

**Example — building a multi-environment config:**
```bash
# Run against the prod signet instance
SIGNET_SERVER=prod.signet:8444 signet sops-key update-config --token "$PROD_TOKEN"
# Updated .sops.yaml
#   environment: prod
#   path_regex:  ^secrets/prod/
#   age key:     age1prod111...

# Run against staging — adds its rule, leaves prod untouched
SIGNET_SERVER=staging.signet:8444 signet sops-key update-config --token "$STAGING_TOKEN"
```

**Resulting `.sops.yaml`:**
```yaml
environments:
  prod: age1prod111...
  staging: age1staging222...

creation_rules:
  - signet_environment: prod
    path_regex: ^secrets/prod/
    age: age1prod111...
  - signet_environment: staging
    path_regex: ^secrets/staging/
    age: age1staging222...
```

The `environments` map is signet-specific and is ignored by the `sops` CLI.
The `signet_environment` annotation on each creation rule is also ignored by
`sops`; it is used by `update-config` to find and update the correct rule
without touching others.

### `signet sops-key prune`

Permanently delete an inactive age key. Any SOPS files that are still encrypted
to this key will fail to sync after pruning.

```
Usage: signet sops-key prune --public-key <age-public-key> [--token <token>]
```

Flags:

| Flag | Required | Description |
|---|---|---|
| `--public-key` | Yes | The age public key string of the key to delete |

**Example:**
```bash
# Only prune after re-encrypting all SOPS files to the new key.
signet sops-key prune --public-key age1abc123def456... --token "$TOKEN"
# Key pruned.
```

---

## `signet secret`

Author SOPS-encrypted secret files in a local repository checkout, without
hand-writing SOPS metadata or learning the `sops` CLI directly. Unlike every
other `signet` command, `secret set`/`secret rm` operate entirely on local
files — they never connect to a signetd instance. Run them from anywhere
inside a checkout that already has a `.sops.yaml` (created by
`signet sops-key update-config`); the repository root and, for
multi-environment repos, the environment are both detected automatically.

Encryption is performed by shelling out to the `sops` binary, which must be
installed and on `PATH` — install it from
[github.com/getsops/sops](https://github.com/getsops/sops#download). This is
deliberate: upstream only guarantees API stability for the `sops` CLI (and its
`decrypt` Go package, used server-side); there is no stable Go package for
encryption to call in-process instead.

### `signet secret set`

Create or update a SOPS-encrypted secret file at the
`<secrets-root>/[<env>/]<namespace>/<service>/<name>.yaml` path convention.

```
Usage: signet secret set <namespace>/<service>/<name> [flags]
```

Flags:

| Flag | Default | Description |
|---|---|---|
| `--value` | `""` | Secret value inline (avoid on shared/logged shells) |
| `--value-file` | `""` | Read the value from a file |
| `--env` | auto-detected | Environment to write under; overrides auto-detection |
| `--secrets-root` | `secrets/` | Secrets directory prefix within the repository |
| `--sops-config` | auto-discovered | Path to `.sops.yaml`; overrides walking up from the current directory |

If neither `--value` nor `--value-file` is given, the value is read from
piped stdin, or from a masked interactive prompt if run in a terminal.

**Environment auto-detection**: the `environments` map in `.sops.yaml`
(written by `signet sops-key update-config`) is consulted when `--env` is not
given — zero entries means a single-environment repo (no environment path
segment), exactly one entry is auto-selected, and more than one requires
`--env` to disambiguate.

**Example — single-environment repo:**
```bash
cd infra-secrets/            # already has a .sops.yaml from update-config
signet secret set payments/api/stripe-key --value sk_live_...
# Wrote encrypted secret: secrets/payments/api/stripe-key.yaml
#   environment: (global)
#
# Next: git add secrets/payments/api/stripe-key.yaml && git commit && git push
# (or 'signet bundle push' if this repository has no remote yet)
```

**Example — multi-environment repo, piped value:**
```bash
printf '%s' "$STRIPE_KEY" | signet secret set payments/api/stripe-key --env prod
# Wrote encrypted secret: secrets/prod/payments/api/stripe-key.yaml
#   environment: prod
```

### `signet secret rm`

Delete a secret file at the same path convention. No decryption is
performed — this is a plain file removal.

```
Usage: signet secret rm <namespace>/<service>/<name> [--env <env>] [--secrets-root <path>] [--sops-config <path>]
```

**Example:**
```bash
signet secret rm payments/api/stripe-key --env prod
# Removed secrets/prod/payments/api/stripe-key.yaml
#
# Next: git add secrets/prod/payments/api/stripe-key.yaml && git commit && git push
```

---

## `signet repo`

Manage git repository registrations for SOPS-encrypted secret syncing.

### `signet repo add`

Register a git repository and receive a GitHub webhook URL and secret.
Optionally creates the webhook in GitHub automatically.

```
Usage: signet repo add --name <name> --repo-url <url> [flags]
```

Flags:

| Flag | Default | Required | Description |
|---|---|---|---|
| `--name` | | Yes | Human alias for the repository |
| `--repo-url` | | Yes | Repository URL, e.g. `git@github.com:org/repo` |
| `--branch` | `main` | | Branch to track |
| `--secrets-path` | `secrets/` | | Directory within the repo containing encrypted secrets |
| `--deploy-key` | | Yes | Path to PEM-encoded SSH deploy key file, or `-` to read from stdin |
| `--setup-webhook` | `false` | | Automatically create the GitHub webhook after registration |
| `--github-token` | | | GitHub PAT for webhook creation (`admin:repo_hook` scope); falls back to `GITHUB_TOKEN` / `GH_TOKEN` env vars |
| `--token` | | | Admin Bearer token |

**Example — manual webhook setup (default):**
```bash
signet repo add \
  --name infra-secrets \
  --repo-url git@github.com:myorg/infra-secrets \
  --branch main \
  --secrets-path secrets/ \
  --deploy-key ./signet_deploy_key \
  --token "$TOKEN"
```

Output:
```
Repository registered.
  ID:             550e8400-e29b-41d4-a716-446655440000
  Webhook URL:    https://signet.example.com/webhook/github/550e8400-...
  Webhook secret: a3f1b2c4d5e6...

This secret will not be shown again.

Configure the webhook in GitHub → Settings → Webhooks.
Run with --setup-webhook to create it automatically.
```

**Example — automatic webhook setup with `gh` CLI:**
```bash
signet repo add \
  --name infra-secrets \
  --repo-url git@github.com:myorg/infra-secrets \
  --deploy-key ./signet_deploy_key \
  --setup-webhook \
  --token "$TOKEN"
```

Output:
```
Repository registered.
  ID:             550e8400-e29b-41d4-a716-446655440000
  Webhook URL:    https://signet.example.com/webhook/github/550e8400-...
  Webhook secret: a3f1b2c4d5e6...

This secret will not be shown again.

Setting up webhook automatically for myorg/infra-secrets...
  Checking for gh CLI... found (/usr/bin/gh).
  Calling `gh api repos/myorg/infra-secrets/hooks`...
  Webhook created.
```

**Example — automatic webhook setup without `gh` CLI:**
```
Setting up webhook automatically for myorg/infra-secrets...
  Checking for gh CLI... not found.
  gh CLI not available, GITHUB_TOKEN is set in environment, calling GitHub's REST API...
  Webhook created.
```

**Automatic webhook creation** (`--setup-webhook`) tries each credential source
in order, logging every step:

1. **`gh` CLI** — if [gh](https://cli.github.com) is installed and authenticated,
   `gh api` is called directly. No extra credentials needed.
2. **`--github-token` flag** — a GitHub personal access token supplied on the
   command line.
3. **`GITHUB_TOKEN` / `GH_TOKEN` environment variables** — checked when neither
   of the above is present.

The token must have the **`admin:repo_hook`** scope (classic PAT) or
**Repository → Administration → write** permission (fine-grained PAT).

If none of the above are available, signet falls back to printing manual
step-by-step instructions — it never fails silently.

Only GitHub repositories are supported for automatic webhook creation. For other
hosts (GitLab, Gitea, etc.) signet prints the manual instructions.

The webhook secret is shown **once only**. If lost, remove and re-register the
repository.

### `signet repo list`

List all registered repositories.

```
Usage: signet repo list [--token <token>]
```

**Example output:**
```
ID                                    NAME            BRANCH  LAST SYNC            SHA
550e8400-e29b-41d4-a716-446655440000  infra-secrets   main    2026-06-24T10:12:00Z a1b2c3d4
```

### `signet repo remove`

Remove a repository registration. This does not delete secrets already synced
into signet's database.

```
Usage: signet repo remove --id <id> [--token <token>]
```

Flags:

| Flag | Required | Description |
|---|---|---|
| `--id` | Yes | Repository ID from `signet repo list` |

### `signet repo sync`

Trigger an immediate full sync of a repository, cloning the current HEAD and
importing all secrets found under the configured `secrets-path`.

```
Usage: signet repo sync --id <id> [--token <token>]
```

Flags:

| Flag | Required | Description |
|---|---|---|
| `--id` | Yes | Repository ID from `signet repo list` |

**Example output:**
```
Sync complete.
  SHA:     a1b2c3d4
  Added:   12
  Updated: 0
  Deleted: 0
```

---

## `signet bundle push`

Push a local git repository directly to signet as a SOPS secret bundle,
without registering it or going through the webhook/reconciler path. Useful
for local testing, CI validation, or one-off syncs where wiring up a
repository registration isn't worth it.

```
Usage: signet bundle push [path] [--secrets-path <path>] [--token <token>]
```

`path` defaults to the current directory. Flags:

| Flag | Default | Description |
|---|---|---|
| `--secrets-path` | `secrets/` | Directory prefix within the repo that holds SOPS-encrypted secrets |

### What it does

1. Opens `path` as a local git repository and resolves `HEAD` — the repo must
   have at least one commit.
2. Walks the files tracked in the `HEAD` commit tree, selecting only `.yaml`
   files under `--secrets-path`.
3. Packages them into an in-memory `tar.gz` (only git-tracked file contents
   are read; nothing on disk outside the repo is touched).
4. Streams the archive to signet's admin API over the same `SyncBundle` RPC
   the webhook/reconciler path uses. Secrets are decrypted server-side with
   signet's age key — only SOPS ciphertext ever leaves your machine.
5. Prints the sync result.

**Example:**
```bash
signet bundle push ./infra-secrets \
  --secrets-path secrets/ \
  --server localhost:8444 \
  --token "$TOKEN"
```

```
Bundled 12 file(s) from HEAD a1b2c3d4e5f6
Sync complete (sha: a1b2c3d4e5f6)
  added:   12
  updated: 0
  deleted: 0
```

If sync errors occur for individual files (e.g. decryption failure for one
secret), they're listed under an `Errors:` section but don't fail the whole
push — other files still sync.

> This bypasses repository registration entirely, so there's no webhook
> secret and no ongoing sync — it's a one-shot push of the current `HEAD`.
> For continuous sync on every push, use [`signet repo add`](#signet-repo-add)
> instead.

---

## Authentication

Every admin command requires a valid Kubernetes ServiceAccount token bound to
the `signet-admin` service account. Generate one with:

```bash
kubectl create token signet-admin -n signet --duration=1h
```

Tokens are short-lived by design. For scripted automation, request a token
from the Kubernetes API in the script rather than storing a long-lived token.

If your `--server` address isn't a loopback address (see
[Transport security](#transport-security) above), the token is only ever sent
over TLS — supply `--ca` if the admin endpoint's certificate isn't in your
system trust store.
