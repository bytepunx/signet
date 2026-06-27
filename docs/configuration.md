# Configuration values

signet stores and distributes non-secret configuration values alongside
encrypted secrets. Common examples: port numbers, feature flags, service
endpoints, log levels — anything a workload needs that does not require SOPS
encryption.

## Overview

```
operator                signet                  workload
────────                ──────                  ────────
edit config YAML        ─→ webhook / bundle
  no encryption needed     parse YAML → JSON
                           store as JSONB
                                           ←── GetServiceConfig (mTLS/SVID)
                                               returns google.protobuf.Struct
                                           ←── WatchServiceConfig (stream)
                                               notifies on any change
```

Config values are stored as structured JSON documents (ingested from plain YAML)
and returned as `google.protobuf.Struct` — the protobuf-native representation of
a JSON object. Every gRPC code generator produces idiomatic access for this type,
and the gRPC-JSON transcoding gateway serves it as raw JSON.

---

## Repository layout

Config files live under a separate root from SOPS-encrypted secrets. Both can
exist in the same repository:

```
secrets/
├── payments/api/stripe-key.yaml    # SOPS-encrypted
└── payments/api/db-password.yaml

config/
├── payments/api.yaml               # plain YAML — all config for payments/api
└── infra/redis.yaml
```

The path convention maps directly to namespace and service:

```
config/<namespace>/<service>.yaml
```

### Environment separation

Use environment-specific subdirectories the same way secrets do — set
`--config-path` to point at the environment subdirectory:

```
config/
├── prod/
│   └── payments/api.yaml
├── staging/
│   └── payments/api.yaml
└── dev/
    └── payments/api.yaml
```

Register the prod instance with `--config-path config/prod/`; the path parser
then reads `<namespace>/<service>.yaml` relative to that root.

---

## File format

A config file is an ordinary YAML mapping. All types supported by YAML are
preserved: strings, numbers, booleans, null, nested objects, arrays.

```yaml
# config/payments/api.yaml
port: 8080
log_level: info
db:
  host: postgres.payments.svc.cluster.local
  port: 5432
  pool_size: 10
feature_flags:
  new_checkout: true
  dark_mode: false
```

Unquoted date literals (e.g. `2023-01-01`) are converted to RFC3339 strings on
ingest. Quote them explicitly if you need the raw string form.

---

## Registering a repository with config sync

Pass `--config-path` when registering a repository. The flag is optional; omit
it to enable secrets-only sync:

```bash
TOKEN=$(kubectl create token signet-admin -n signet --duration=1h)

signet repo add \
  --name         my-repo \
  --repo-url     git@github.com:myorg/my-repo \
  --secrets-path secrets/ \
  --config-path  config/ \
  --deploy-key   ./signet_deploy_key \
  --setup-webhook \
  --token        "$TOKEN"
```

The two paths are independent: a push that changes only a config file will not
trigger secret decryption, and vice versa.

---

## Triggering a sync

The webhook handles incremental updates automatically. For a full sync
(initial setup or after a misconfigured webhook period):

```bash
signet repo sync --id "$REPO_ID" --token "$TOKEN"
```

Output:
```
Synced.
  SHA:            a1b2c3d4...
  Secrets added:  0
  Configs synced: 1
```

---

## Retrieving config from a workload

### Full service config

```go
// Go client using go-spiffe for mTLS
source, _ := workloadapi.NewX509Source(ctx)
defer source.Close()

creds := grpccredentials.MTLSClientCredentials(source, source,
    tlsconfig.AuthorizeID(spiffeid.RequireIDFromString("spiffe://cluster.local/signet")))

conn, _ := grpc.Dial("signet.signet.svc.cluster.local:8443", grpc.WithTransportCredentials(creds))
client := signetv1.NewSecretsServiceClient(conn)

resp, _ := client.GetServiceConfig(ctx, &signetv1.GetServiceConfigRequest{
    Namespace: "payments",
    Service:   "api",
})
// resp.Values is a *structpb.Struct
port := resp.Values.Fields["port"].GetNumberValue() // → 8080.0
dbHost := resp.Values.Fields["db"].GetStructValue().Fields["host"].GetStringValue()
```

### Single key (dot-path)

```go
resp, _ := client.GetConfig(ctx, &signetv1.GetConfigRequest{
    Namespace: "payments",
    Service:   "api",
    Key:       "db.host",
})
// resp.Value is a *structpb.Value
host := resp.Value.GetStringValue() // → "postgres.payments.svc.cluster.local"
```

Dot-path navigation supports arbitrary depth: `"feature_flags.new_checkout"`,
`"db.pool_size"`, etc.

### Watching for changes

`WatchServiceConfig` streams the full config document on connection, then
re-sends it whenever any key changes. Clients are responsible for acting on
updates — a rolling restart, a config reload, or a hot-reload depending on the
workload's needs.

```go
stream, _ := client.WatchServiceConfig(ctx, &signetv1.WatchServiceConfigRequest{
    Namespace: "payments",
    Service:   "api",
})
for {
    resp, err := stream.Recv()
    if err != nil { break }
    switch resp.EventType {
    case signetv1.WatchServiceConfigResponse_EVENT_TYPE_UPDATED:
        reloadConfig(resp.Values)
    case signetv1.WatchServiceConfigResponse_EVENT_TYPE_DELETED:
        // Config file was removed from the repository
    }
}
```

---

## Retrieving the full service bundle

`GetServiceBundle` merges config and secrets into a single call. Config keys sit
at the top level; all secrets appear base64-encoded under the reserved `"secrets"`
key. This is the primary API for workloads that need both config and secrets at
startup.

```go
resp, _ := client.GetServiceBundle(ctx, &signetv1.GetServiceBundleRequest{
    Namespace: "payments",
    Service:   "api",
})
// Top-level config
port := resp.Bundle.Fields["port"].GetNumberValue()
// Secrets under "secrets" key, base64-encoded
secsField := resp.Bundle.Fields["secrets"].GetStructValue()
stripeKeyB64 := secsField.Fields["stripe-key"].GetStringValue()
stripeKey, _ := base64.StdEncoding.DecodeString(stripeKeyB64)
```

The `"secrets"` key is reserved: if a config YAML file contains a top-level
`secrets` key, it is overwritten by the secrets map. signet logs a warning when
this collision is detected during a sync.

### Watching for any change

`WatchServiceBundle` is a notification-only stream — no values are included in
change events. The intended pattern is coordinated shutdown: when a notification
arrives, the workload finishes its current requests, shuts down cleanly, and
re-fetches configuration on the next startup via `GetServiceBundle`.

No event is sent on initial connection; a workload should always call
`GetServiceBundle` on startup before opening the watch stream.

```go
stream, _ := client.WatchServiceBundle(ctx, &signetv1.WatchServiceBundleRequest{
    Namespace: "payments",
    Service:   "api",
})
for {
    resp, err := stream.Recv()
    if err != nil { break }
    switch resp.EventType {
    case signetv1.WatchServiceBundleResponse_EVENT_TYPE_CHANGED:
        // A secret or config value has changed.
        // Initiate a coordinated shutdown and let the process restart fresh.
        initiateGracefulShutdown()
    }
}
```

This design avoids the complexity of live-patching application state and ensures
that every running instance always reflects a single consistent snapshot.

---

## Access control

The [convention-first policy model](policies.md) applies to config identically
to secrets: a workload whose SPIFFE ID encodes `ns/<namespace>/sa/<service>` can
call `GetServiceConfig` and `WatchServiceConfig` for that same namespace/service
without any explicit policy. Cross-service or cross-namespace access requires an
explicit policy.

---

## Operator workflow

```bash
# Edit config directly — no encryption tooling required
vim config/payments/api.yaml

git add config/payments/api.yaml
git commit -m "increase db pool size for payments/api"
git push
# signet picks up the change via webhook within seconds
```

Config files are diff-friendly, auditable via git history, and require no
operator tooling beyond a text editor and git.

---

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| Config not appearing after push | Verify `--config-path` was set during `repo add`; check webhook delivery in GitHub; confirm file is at `<config-path>/<namespace>/<service>.yaml` |
| `GetServiceConfig` returns `NOT_FOUND` | Config not yet synced; run `signet repo sync` to trigger a full sync |
| `GetServiceConfig` returns `PERMISSION_DENIED` | Workload SPIFFE ID does not match namespace/service; see [policies.md](policies.md) |
| `GetConfig` returns `NOT_FOUND` for a key | Key path is wrong or the key was removed from the YAML file |
| `GetServiceBundle` has empty `secrets` | No secrets stored for this namespace/service; run a full sync |
| `WatchServiceBundle` never fires | Confirm secrets and config path are both registered; check webhook delivery |
| Secret value in bundle is wrong length | Client must base64-decode the string value before use |
| `configs_synced: 0` after full sync | Config path does not exist in the repository or no `.yaml` files match the expected two-component path |
