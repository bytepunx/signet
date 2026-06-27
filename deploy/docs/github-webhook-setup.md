# GitHub Webhook Setup for signet

signet syncs secrets from a git repository by listening for GitHub push events.
This guide covers registering the repository with signet and configuring the
webhook in GitHub.

## Prerequisites

- signet running and unsealed
- `signet` CLI configured (`signet config set server http://localhost:8444`)
- signet's webhook port (`:8445`) reachable from GitHub's webhook IPs
  - For local development, use [smee.io](https://smee.io) or `ngrok` to proxy
- An SSH deploy key with **read-only** access to the repository
- Secrets already encrypted with SOPS (see `sops-setup.md`)

## 1. Generate a deploy key

```bash
# Generate a dedicated read-only deploy key for signet
ssh-keygen -t ed25519 -C "signet-deploy" -f signet_deploy_key -N ""
```

Add the **public key** (`signet_deploy_key.pub`) to your repository:
- GitHub → Repository → Settings → Deploy keys → Add deploy key
- Title: `signet`
- Key: paste contents of `signet_deploy_key.pub`
- Leave "Allow write access" **unchecked**

## 2. Register the repository with signet

```bash
signet repo add \
  --name        infra-secrets \
  --repo-url    git@github.com:your-org/infra-secrets \
  --branch      main \
  --secrets-path secrets/ \
  --deploy-key  ./signet_deploy_key
```

Output:
```
Repository registered.
  ID:             550e8400-e29b-41d4-a716-446655440000
  Webhook URL:    https://signet.example.com/webhook/github/550e8400-...
  Webhook secret: a3f1b2c4d5e6...

Configure the webhook in GitHub → Settings → Webhooks.
This secret will not be shown again.
```

**Save the webhook secret** — it is shown only once. If you lose it, remove the
repository (`signet repo remove --id <id>`) and register again.

Delete the deploy key file from disk after registration:

```bash
rm signet_deploy_key  # signet has stored the encrypted copy
```

## 3. Configure the webhook in GitHub

1. Go to your repository on GitHub
2. **Settings** → **Webhooks** → **Add webhook**
3. Fill in:

   | Field | Value |
   |---|---|
   | Payload URL | The webhook URL from step 2 (e.g. `https://signet.example.com/webhook/github/550e8400-...`) |
   | Content type | `application/json` |
   | Secret | The webhook secret from step 2 |
   | SSL verification | Enable (signet's webhook port must have TLS in production) |
   | Which events? | **Just the `push` event** |
   | Active | ✓ |

4. Click **Add webhook**

GitHub will immediately send a ping event. The ping returns 200 OK (it is
not a push event and signet ignores it). If you see a red ✗, check that the
webhook URL is reachable from GitHub's IP ranges.

## 4. Trigger an initial full sync

The webhook only processes future pushes. To import secrets that already exist
in the repository, trigger a full sync manually:

```bash
REPO_ID=550e8400-e29b-41d4-a716-446655440000

signet repo sync --id "$REPO_ID"
```

Output:
```
Sync complete.
  SHA:     a1b2c3d4
  Added:   12
  Updated: 0
  Deleted: 0
```

## 5. Verify ongoing sync

After pushing a new commit that changes a secret file, check the delivery log
in GitHub (Settings → Webhooks → your webhook → Recent Deliveries). A green
checkmark indicates signet received and processed the event.

You can also check `signet repo list` — the `last-sync` timestamp and SHA
update after each successful sync.

## Sealed-state behaviour

If signet is sealed when a webhook arrives, it returns HTTP 503 with a
`Retry-After: 30` header. GitHub will automatically retry the delivery for up
to 72 hours. Once signet is unsealed, the next retry will succeed and the
missed events will be replayed. The periodic reconciler (runs every 5 minutes
by default) provides an additional safety net.

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| 404 on webhook URL | Repository ID in URL is wrong, or signet's webhook port is not reachable |
| 401 Unauthorized | Webhook secret mismatch — re-register the repository |
| 503 Service Unavailable | signet is sealed — unseal and GitHub will retry |
| Secrets not appearing | Check `signet repo list` for last-sync time; check SOPS file format (must have top-level `value` key) and path structure (`secrets/<ns>/<svc>/<name>.yaml`) |
