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

When neither `--token` nor `--token-file` is supplied, signet attempts to use
a token from the environment variable `SIGNET_TOKEN`.

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
| `--namespace` | `signet` | Kubernetes namespace where the Secret is created |
| `--key-secret` | `signet-master-key` | Name of the Kubernetes Secret to create or reuse |
| `--kube-context` | `""` | kubeconfig context to use; defaults to current context |
| `--force` | `false` | Overwrite an existing Secret and re-unseal even if already unsealed |
| `--dry-run` | `false` | Print what would happen without creating any resources or contacting the server |

### What it does

1. Checks whether a Secret named `--key-secret` already exists in `--namespace`.
2. If not found, generates 32 random bytes as the master key and stores them in a new Secret.
3. Reads the key back from the Secret.
4. Sends the key to signetd via `signet unseal key`.
5. Prints a confirmation with the key source and resulting server state.

### Example — fresh cluster (Secret not found)

```
$ signet init --server localhost:8444 --token "$TOKEN"
Secret signet-master-key not found in namespace signet — generating new key.
Created Secret signet/signet-master-key.
Server unsealed.
```

### Example — re-running when already unsealed

```
$ signet init --server localhost:8444 --token "$TOKEN"
Secret signet-master-key found in namespace signet.
Server is already unsealed. Use --force to unseal again.
```

### `--force` semantics

When `--force` is supplied alongside an existing Secret, `signet init` reads the
existing key (it does **not** regenerate it) and re-submits it to the server.
This is useful after a pod restart when auto-unseal is not enabled. To replace
the key entirely, delete the Secret manually first.

### `--dry-run` for CI validation

`--dry-run` performs all checks (context resolution, namespace lookup, server
reachability) and prints every action it would take, without writing any
Kubernetes resources or calling the unseal RPC. Use this in CI to verify the
command would succeed without side effects.

```bash
signet init --dry-run
# [dry-run] Would create Secret signet/signet-master-key.
# [dry-run] Would unseal server at localhost:8444.
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

## Authentication

Every admin command requires a valid Kubernetes ServiceAccount token bound to
the `signet-admin` service account. Generate one with:

```bash
kubectl create token signet-admin -n signet --duration=1h
```

Tokens are short-lived by design. For scripted automation, request a token
from the Kubernetes API in the script rather than storing a long-lived token.
