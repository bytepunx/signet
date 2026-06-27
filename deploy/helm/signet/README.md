# signet Helm chart

Deploys [signet](https://github.com/bytepunx/signet) — a SPIFFE/SPIRE-native
configuration and secrets management service for Kubernetes.

## Prerequisites

| Dependency | Required version | Notes |
|---|---|---|
| Kubernetes | ≥ 1.25 | Policy/v1 PDB; networking.k8s.io/v1 Ingress |
| Helm | ≥ 3.12 | OCI registry support; values schema validation |
| SPIRE | any | Server and agent must already be running; trust domain must match `signet.trustDomain` |
| CockroachDB | ≥ 24 | External cluster recommended for production; in-chart dev instance available |

SPIRE is **not** included in this chart. Deploy it first:
```bash
helm install spire oci://ghcr.io/spiffe/helm-charts-hardened/spire \
  --set global.spire.trustDomain=cluster.local \
  --namespace spire-system --create-namespace
```

## Quick start

```bash
# 1. Install the chart (dev mode: in-cluster CockroachDB, direct-key unseal)
helm install signet oci://ghcr.io/bytepunx/charts/signet \
  --set signet.trustDomain=cluster.local \
  --set cockroachdb.enabled=true \
  --set signet.dbConnString="postgresql://root@signet-cockroachdb.default.svc.cluster.local:26257/signet?sslmode=disable" \
  --set signet.auditChainKey="$(openssl rand -hex 32)" \
  --namespace signet --create-namespace

# 2. Forward the admin port (never exposed externally)
kubectl port-forward -n signet svc/signet 8444:8444 &

# 3. Generate an unseal token and unseal
TOKEN=$(kubectl create token signet-admin -n signet --duration=1h)
signet config set server http://localhost:8444
signet unseal --key "$(openssl rand -hex 32)" --token "$TOKEN"

# 4. Verify
signet status
```

## Required values

| Value | Description |
|---|---|
| `signet.trustDomain` | SPIFFE trust domain — must match your SPIRE server |

Either `signet.existingSecret` (recommended) or both `signet.dbConnString` and
`signet.auditChainKey` must be set.

## Key values reference

| Value | Default | Description |
|---|---|---|
| `signet.trustDomain` | `""` | **Required.** SPIFFE trust domain |
| `signet.existingSecret` | `""` | Name of a pre-created Secret containing `SIGNET_DB_CONN_STRING` and `SIGNET_AUDIT_CHAIN_KEY` |
| `signet.webhookAddr` | `":8445"` | Webhook listener address; set to `""` to disable |
| `signet.webhookBaseURL` | `""` | Public base URL returned by `signet repo add` |
| `signet.shamir.shares` | `0` | Shamir shares (0 = direct-key mode) |
| `signet.shamir.threshold` | `0` | Shamir threshold |
| `replicaCount` | `1` | Pod replicas (see scaling note below) |
| `global.image.registry` | `""` | Override registry for air-gapped installs |
| `ingress.enabled` | `false` | Expose webhook port via Ingress |
| `cockroachdb.enabled` | `false` | Deploy a single-node CockroachDB (dev only) |
| `networkPolicy.enabled` | `true` | Restrict ingress/egress with NetworkPolicy |

## Auto-unseal

> **Trust boundary:** enabling auto-unseal stores the master key in a
> Kubernetes Secret. Any principal with cluster-admin access to the namespace
> can read it — this is a weaker guarantee than Shamir. Use only where that
> boundary is acceptable (dev clusters, single-operator setups).

Enable auto-unseal in your values:

```yaml
autoUnseal:
  enabled: true
  secretName: signet-master-key  # default
```

The chart creates a namespaced `Role` and `RoleBinding` so signetd can read
only the named Secret, and sets `SIGNET_KUBE_UNSEAL_SECRET` in the ConfigMap
so signetd unseals itself on startup.

### Combined workflow

Use `signet init` to create the Secret on first deploy, then rely on
`autoUnseal` for restarts:

```bash
# First deploy — create key and unseal manually
TOKEN=$(kubectl create token signet-admin -n signet --duration=1h)
signet init --server localhost:8444 --token "$TOKEN"

# Upgrade to enable auto-unseal for future pod restarts
helm upgrade signet oci://ghcr.io/bytepunx/charts/signet \
  --set autoUnseal.enabled=true
```

> **Production note:** for environments with multiple operators or compliance
> requirements, use Shamir unseal (`signet.shamir.shares` / `threshold`) and
> distribute key shares to separate people. For additional hardening, consider
> storing the Secret via [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets)
> or a secrets operator with strict RBAC rather than a plain Kubernetes Secret.

---

## Production checklist

- [ ] Use `signet.existingSecret` — create the secret with external-secrets or sealed-secrets, never commit credentials
- [ ] Set `cockroachdb.enabled: false`; supply a connection string to a production CockroachDB cluster
- [ ] Enable Shamir unseal (`signet.shamir.shares ≥ 3`, `threshold ≥ 2`) and distribute shares to separate key holders
- [ ] Enable Ingress with TLS if using GitHub webhooks
- [ ] Set `signet.webhookBaseURL` to the public webhook URL
- [ ] Set `replicaCount: 1` (scaling limitation: master key is in-process)

## Scaling

signet holds the decrypted master key in memory. Running more than one replica
requires each pod to be unsealed independently and does **not** provide active
failover — traffic will hit only one pod at a time via ClusterIP. The HPA is
provided for future use when a shared key store is supported; leave
`autoscaling.enabled: false` for now.

## Air-gapped installs

Mirror the image to your private registry, then override the registry:

```bash
# Mirror
crane copy ghcr.io/bytepunx/signetd:v1.0.0 myregistry.internal/bytepunx/signetd:v1.0.0

# Install
helm install signet oci://ghcr.io/bytepunx/charts/signet \
  --set global.image.registry=myregistry.internal \
  --set signet.trustDomain=cluster.local \
  ...
```

## Helm test

After install, run `helm test signet -n signet` to verify signet's gRPC port
is reachable within the cluster. The test pod is cleaned up on success.
