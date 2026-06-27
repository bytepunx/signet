# signet installation and configuration reference

## Prerequisites

| Requirement | Notes |
|---|---|
| Kubernetes ≥ 1.25 | Any distribution; GKE, EKS, AKS, k3s all work |
| Helm ≥ 3.12 | OCI chart support |
| SPIRE | Deployed and healthy; agent DaemonSet running on all nodes |
| CockroachDB | External cluster recommended for production |
| kubectl access | To create secrets and port-forward the admin endpoint |

---

## Installing the chart

```bash
helm install signet oci://ghcr.io/bytepunx/charts/signet \
  --namespace signet --create-namespace \
  --set signet.trustDomain=<your-trust-domain> \
  --set signet.existingSecret=signet-secrets
```

All configuration is supplied via Helm values. The reference below covers every
supported value.

---

## Helm values reference

### Top-level

| Value | Default | Description |
|---|---|---|
| `global.image.registry` | `""` | Override image registry for air-gapped installs (e.g. `myregistry.internal`) |
| `replicaCount` | `1` | Number of signetd pods; see [Scaling](#scaling) |
| `image.registry` | `ghcr.io` | Image registry |
| `image.repository` | `bytepunx/signetd` | Image repository |
| `image.tag` | `""` | Image tag; defaults to chart `appVersion` |
| `image.pullPolicy` | `IfNotPresent` | Kubernetes image pull policy |
| `nameOverride` | `""` | Override chart name component |
| `fullnameOverride` | `""` | Override the full resource name prefix |

### `signet.*`

| Value | Default | Required | Description |
|---|---|---|---|
| `signet.trustDomain` | `""` | **Yes** | SPIFFE trust domain — must match your SPIRE server |
| `signet.workloadAddr` | `":8443"` | | gRPC workload listener address |
| `signet.adminAddr` | `"127.0.0.1:8444"` | | Admin gRPC listener; bound to localhost, never exposed |
| `signet.webhookAddr` | `":8445"` | | GitHub webhook listener; set to `""` to disable |
| `signet.webhookBaseURL` | `""` | | Public base URL returned by `signet repo add` |
| `signet.drainTimeout` | `"30s"` | | Graceful shutdown drain timeout |
| `signet.spireSocket` | `"unix:///run/spire/sockets/agent.sock"` | | SPIRE agent socket path |
| `signet.kubeAudiences` | `"signet"` | | Comma-separated SA token audiences for the admin endpoint |
| `signet.existingSecret` | `""` | One of these | Name of a pre-created Secret containing credentials |
| `signet.dbConnString` | `""` | One of these | Database connection string (ignored if `existingSecret` is set) |
| `signet.auditChainKey` | `""` | One of these | 64-hex-char (32-byte) HMAC chain key (ignored if `existingSecret` is set) |

### `signet.shamir.*` — Shamir unseal mode {#shamir-unseal}

In Shamir mode, the master key is split into N shares. Any K shares reconstruct
it. Set `shares > 0` and `threshold ≥ 2` to enable; leave both at `0` for
direct-key mode.

| Value | Default | Description |
|---|---|---|
| `signet.shamir.shares` | `0` | Total number of shares to distribute |
| `signet.shamir.threshold` | `0` | Minimum shares required to unseal |
| `signet.shamir.shareTimeout` | `"30m"` | Duration a submitted share remains valid; resets to sealed if not completed |

**Example — 3-of-5 Shamir:**
```yaml
signet:
  shamir:
    shares: 5
    threshold: 3
    shareTimeout: 15m
```

After install, generate shares and distribute them to key holders:
```bash
TOKEN=$(kubectl create token signet-admin -n signet --duration=1h)
signet unseal generate-shares --shares 5 --threshold 3 --token "$TOKEN"
# Outputs 5 hex-encoded shares. Distribute each to a separate person.
```

Each key holder submits their share independently:
```bash
signet unseal share --share <hex> --token "$TOKEN"
```

Once the threshold is met, signet unseals automatically.

### `autoUnseal.*` — Auto-unseal via Kubernetes Secret {#auto-unseal}

In auto-unseal mode, signetd reads the master key from a Kubernetes Secret at
startup and unseals itself without any operator intervention. This is convenient
for dev clusters and single-operator setups, but weakens the security boundary
compared to Shamir: **any principal with cluster-admin access to the namespace
can read the Secret and therefore read the master key.** Do not use this mode
in production multi-operator environments.

**When to use:**
- Local or dev clusters where convenience outweighs key custody requirements
- Single-operator setups where the cluster-admin trust boundary is acceptable
- Automated CI pipelines that need a sealed-then-unsealed signet instance

**When NOT to use:**
- Production clusters with multiple operators or compliance requirements
- Any environment where cluster-admin access is not equivalent to full key access
- Setups where you need the security guarantee that no single person can unseal alone

#### Helm values

| Value | Default | Description |
|---|---|---|
| `autoUnseal.enabled` | `false` | Enable auto-unseal from a Kubernetes Secret |
| `autoUnseal.secretName` | `signet-master-key` | Name of the Secret containing the master key |

Enable auto-unseal:

```yaml
autoUnseal:
  enabled: true
  secretName: signet-master-key
```

When enabled, the Helm chart creates:
- A namespaced `Role` granting `get` on the named Secret
- A `RoleBinding` attaching that Role to the signetd ServiceAccount
- Sets `SIGNET_KUBE_UNSEAL_SECRET` in the signetd ConfigMap so signetd reads
  and applies the key on startup

#### Initial setup with `signet init`

Auto-unseal requires the Secret to exist before signetd starts. Use
`signet init` to create it on first deploy, then enable `autoUnseal` for
subsequent restarts:

```bash
# 1. Deploy signet without auto-unseal, then create the key Secret:
TOKEN=$(kubectl create token signet-admin -n signet --duration=1h)
signet init --server localhost:8444 --token "$TOKEN"

# 2. Enable auto-unseal so future pod restarts unseal automatically:
helm upgrade signet oci://ghcr.io/bytepunx/charts/signet \
  --set autoUnseal.enabled=true
```

#### Alternative: manual unseal without auto-unseal

If you created the Secret with `signet init` but prefer not to use auto-unseal,
signetd will start sealed and you unseal manually on each restart:

```bash
TOKEN=$(kubectl create token signet-admin -n signet --duration=1h)
signet init --force --server localhost:8444 --token "$TOKEN"
```

`--force` reads the existing Secret and re-submits the key without regenerating
it.

---

### `serviceAccount.*` and `adminServiceAccount.*`

| Value | Default | Description |
|---|---|---|
| `serviceAccount.create` | `true` | Create the signetd ServiceAccount (needs TokenReview RBAC) |
| `serviceAccount.name` | `""` | Override SA name; defaults to the full release name |
| `serviceAccount.annotations` | `{}` | Annotations on the SA (e.g. for IRSA or Workload Identity) |
| `adminServiceAccount.create` | `true` | Create the `signet-admin` SA for operator token generation |
| `adminServiceAccount.name` | `"signet-admin"` | Name of the admin SA |

### `service.*`

| Value | Default | Description |
|---|---|---|
| `service.type` | `ClusterIP` | Kubernetes Service type |
| `service.port` | `8443` | Service port for the workload gRPC endpoint |

### `ingress.*` — Webhook Ingress

Required when GitHub needs to reach the webhook from the public internet.

| Value | Default | Description |
|---|---|---|
| `ingress.enabled` | `false` | Enable Ingress for the webhook port |
| `ingress.className` | `""` | IngressClass name (e.g. `nginx`) |
| `ingress.annotations` | `{}` | Annotations (e.g. cert-manager issuer) |
| `ingress.hosts` | `[]` | List of `{host, paths[]}` |
| `ingress.tls` | `[]` | TLS configuration |

**Example — nginx + cert-manager:**
```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: signet.example.com
      paths:
        - path: /webhook
          pathType: Prefix
  tls:
    - secretName: signet-tls
      hosts:
        - signet.example.com
```

### `autoscaling.*`

Disabled by default. See [Scaling](#scaling) before enabling.

| Value | Default | Description |
|---|---|---|
| `autoscaling.enabled` | `false` | Enable HorizontalPodAutoscaler |
| `autoscaling.minReplicas` | `1` | Minimum replicas |
| `autoscaling.maxReplicas` | `3` | Maximum replicas |
| `autoscaling.targetCPUUtilizationPercentage` | `80` | CPU target for scaling |

### `resources.*`

```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    memory: 256Mi
```

CPU is not limited by default because short spikes during decryption are
normal and a CPU limit would cause unnecessary throttling.

### `networkPolicy.enabled`

Default `true`. Restricts ingress to the gRPC port (8443) from within the
cluster, and webhook port (8445) when `signet.webhookAddr` is set. Egress to
the database and SPIRE agent socket is allowed; all other egress is blocked.

### `cockroachdb.*` — In-cluster CockroachDB (dev only)

| Value | Default | Description |
|---|---|---|
| `cockroachdb.enabled` | `false` | Deploy a single-node CockroachDB StatefulSet |
| `cockroachdb.replicas` | `1` | CockroachDB replicas (1 for dev, 3 for minimal HA) |
| `cockroachdb.image` | `cockroachdb/cockroach:v24.3.0` | CockroachDB image |
| `cockroachdb.storage` | `"10Gi"` | PVC size |
| `cockroachdb.storageClass` | `""` | StorageClass; cluster default if empty |

For production, use the [CockroachDB Kubernetes Operator](https://www.cockroachlabs.com/docs/stable/kubernetes-overview.html).

---

## Environment variable reference

These are injected via the ConfigMap and Secret; the Helm chart manages them
automatically. You only need these if running signetd outside Kubernetes.

| Variable | Required | Description |
|---|---|---|
| `SIGNET_TRUST_DOMAIN` | Yes | SPIFFE trust domain |
| `SIGNET_WORKLOAD_ADDR` | | gRPC workload listener (default `:8443`) |
| `SIGNET_ADMIN_ADDR` | | Admin listener (default `127.0.0.1:8444`) |
| `SIGNET_WEBHOOK_ADDR` | | Webhook listener; empty disables (default `:8445`) |
| `SIGNET_WEBHOOK_BASE_URL` | | Public webhook base URL |
| `SIGNET_DRAIN_TIMEOUT` | | Graceful shutdown timeout (default `30s`) |
| `SIGNET_SPIRE_SOCKET` | | SPIRE agent socket (default `unix:///run/spire/sockets/agent.sock`) |
| `SIGNET_SHAMIR_SHARES` | | Shamir shares; 0 = direct-key mode |
| `SIGNET_SHAMIR_THRESHOLD` | | Shamir threshold |
| `SIGNET_SHAMIR_SHARE_TIMEOUT` | | Share validity window (default `30m`) |
| `SIGNET_KUBE_AUDIENCES` | | Comma-separated token audiences (default `signet`) |
| `SIGNET_DB_CONN_STRING` | Yes | CockroachDB/PostgreSQL connection string |
| `SIGNET_AUDIT_CHAIN_KEY` | Yes | 64 hex-char HMAC chain key |

---

## SPIRE ClusterSpiffeID examples

Register entries in SPIRE so signet workloads receive SVIDs. The SPIFFE ID
format used in signet policies is:

```
spiffe://<trust-domain>/ns/<namespace>/sa/<service-account>
```

### Register a single workload

```bash
kubectl exec -n spire-system deployment/spire-server -- \
  /opt/spire/bin/spire-server entry create \
    -spiffeID spiffe://cluster.local/ns/payments/sa/api \
    -parentID spiffe://cluster.local/spire/agent/k8s_sat/cluster.local/node \
    -selector k8s:ns:payments \
    -selector k8s:sa:api
```

### Using ClusterSPIFFEID (SPIRE operator)

```yaml
apiVersion: spire.spiffe.io/v1alpha1
kind: ClusterSPIFFEID
metadata:
  name: payments-api
spec:
  spiffeIDTemplate: "spiffe://cluster.local/ns/{{ .PodMeta.Namespace }}/sa/{{ .PodSpec.ServiceAccountName }}"
  podSelector:
    matchLabels:
      app.kubernetes.io/name: api
  namespaceSelector:
    matchLabels:
      kubernetes.io/metadata.name: payments
```

---

## Scaling

signet currently holds the decrypted master key in a single in-process
`memguard.LockedBuffer`. Running multiple replicas means:

- Each pod must be unsealed independently after a restart
- The ClusterIP service distributes traffic; pods that are sealed will return
  `UNAVAILABLE` errors until unsealed
- There is no active-active key replication between pods

For high availability with multiple pods, use Shamir unseal mode and automate
share submission via a secrets manager (e.g. Vault or AWS Secrets Manager holds
the shares and submits them on pod startup via an init container).

---

## CockroachDB sizing (production)

| Workload | Nodes | CPU/node | Memory/node | Storage/node |
|---|---|---|---|---|
| Small (< 10k secrets) | 3 | 2 | 8 GiB | 100 GiB |
| Medium (10k–100k secrets) | 3 | 4 | 16 GiB | 500 GiB |
| Large (> 100k secrets) | 5 | 8 | 32 GiB | 1 TiB |

Enable the CockroachDB Prometheus endpoint and alert on:
- `capacity_available / capacity` < 20%
- `sys_cpu_user_percent` > 70% sustained
- Replication lag (under-replicated ranges)
