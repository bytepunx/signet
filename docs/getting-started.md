# Getting started with signet

This guide walks a new operator through deploying signet on an existing
Kubernetes cluster, from zero to a workload successfully retrieving a secret.

**Time:** ~30 minutes  
**Prerequisite:** a working Kubernetes cluster with `kubectl` and `helm` access

---

## Step 1 — Deploy SPIRE

signet requires SPIRE for workload identity. Skip this step if SPIRE is already
running in your cluster.

```bash
helm install spire oci://ghcr.io/spiffe/helm-charts-hardened/spire \
  --set global.spire.trustDomain=cluster.local \
  --namespace spire-system --create-namespace

kubectl rollout status -n spire-system daemonset/spire-agent
```

Verify SPIRE is healthy:
```bash
kubectl exec -n spire-system daemonset/spire-agent -- \
  /opt/spire/bin/spire-agent healthcheck
```

---

## Step 2 — Deploy signet

```bash
# Create namespace
kubectl create namespace signet

# Generate the audit chain key (32 random bytes as hex)
AUDIT_KEY=$(openssl rand -hex 32)

# Create the signet secret out of band.
# For production, use external-secrets or sealed-secrets instead.
kubectl create secret generic signet-secrets -n signet \
  --from-literal=SIGNET_DB_CONN_STRING="postgresql://root@localhost:26257/signet?sslmode=disable" \
  --from-literal=SIGNET_AUDIT_CHAIN_KEY="${AUDIT_KEY}"

# Install signet (dev mode: in-cluster CockroachDB)
helm install signet oci://ghcr.io/bytepunx/charts/signet \
  --namespace signet \
  --set signet.trustDomain=cluster.local \
  --set signet.existingSecret=signet-secrets \
  --set cockroachdb.enabled=true \
  --set signet.dbConnString="postgresql://root@signet-cockroachdb.signet.svc.cluster.local:26257/signet?sslmode=disable"

kubectl rollout status -n signet deployment/signet
```

---

## Step 3 — Connect the CLI

Download the `signet` CLI binary for your OS from the
[latest release](https://github.com/bytepunx/signet/releases/latest) and put
it on your `$PATH`.

Open a port-forward to the admin endpoint (it is never exposed externally):

```bash
kubectl port-forward -n signet svc/signet 8444:8444 &
```

Configure the CLI:

```bash
signet config set server http://localhost:8444
signet status
# Output: state=sealed
```

---

## Step 4 — Unseal

signet starts sealed — the master key is not in memory and no secrets can be
retrieved until you unseal it.

### `signet init` — recommended for dev clusters

`signet init` generates a master key, stores it in a Kubernetes Secret, and
unseals the server in one command:

```bash
TOKEN=$(kubectl create token signet-admin -n signet --duration=1h)
signet init --server localhost:8444 --token "$TOKEN"
# Secret signet-master-key not found in namespace signet — generating new key.
# Created Secret signet/signet-master-key.
# Server unsealed.

signet status
# Output: state=unsealed
```

> **Trust boundary:** the master key is stored in a Kubernetes Secret. Any
> principal with cluster-admin access can read it. This is acceptable for dev
> clusters and single-operator setups.

### Shamir mode (production alternative)

For production multi-operator environments, use Shamir unseal instead. The
key is split into N shares held by separate people; no single person (or
compromised cluster-admin) can unseal alone. See
[installation.md](installation.md#shamir-unseal) for setup — it requires
configuring `signet.shamir.shares` and `signet.shamir.threshold` in the Helm
values before deploying.

---

## Step 5 — Register a SPIRE workload entry

signet uses the workload's SPIFFE ID (from its X.509 SVID) to identify it.
Register an entry in SPIRE for the workload that will read secrets:

```bash
kubectl exec -n spire-system deployment/spire-server -- \
  /opt/spire/bin/spire-server entry create \
    -spiffeID spiffe://cluster.local/ns/payments/sa/api \
    -parentID spiffe://cluster.local/spire/agent/k8s_sat/cluster.local/node \
    -selector k8s:ns:payments \
    -selector k8s:sa:api
```

This grants the `api` service account in the `payments` namespace the SVID
`spiffe://cluster.local/ns/payments/sa/api`.

---

## Step 6 — Create an access policy

signet policies control which workload SVIDs can read which secrets.

```bash
TOKEN=$(kubectl create token signet-admin -n signet --duration=5m)

# Allow the payments/api workload to read any secret in the payments namespace.
signet policy create \
  --spiffe-id "spiffe://cluster.local/ns/payments/sa/api" \
  --namespace payments \
  --service api \
  --token "$TOKEN"
```

---

## Step 7 — Push an encrypted secret

Follow [secrets-workflow.md](secrets-workflow.md) for the full SOPS setup. The
abbreviated steps are:

```bash
# Get the signet age public key
TOKEN=$(kubectl create token signet-admin -n signet --duration=5m)
signet sops-key get --token "$TOKEN"
# Public key:  age1abc123...

# In your secrets repository, create .sops.yaml
cat > .sops.yaml <<EOF
creation_rules:
  - path_regex: secrets/.*\.yaml$
    age: age1abc123...
EOF

# Create and encrypt a secret
mkdir -p secrets/payments/api
echo "value: sk_live_abc123" > secrets/payments/api/stripe-key.yaml
sops --encrypt --in-place secrets/payments/api/stripe-key.yaml

# Push — the reconciler or webhook will pick it up
git add . && git commit -m "add stripe key" && git push
```

---

## Step 8 — Verify secret retrieval

A workload with the correct SPIFFE ID can now retrieve the secret via the gRPC
API. From a pod in the `payments` namespace with service account `api`:

```bash
# The go-spiffe library handles SVID fetching automatically.
# Using grpcurl to test manually:
grpcurl \
  -spiffe \
  -spiffe-bundle /run/spire/bundle.crt \
  -spiffe-svid /run/spire/svid.pem \
  -spiffe-key  /run/spire/svid_key.pem \
  signet.signet.svc.cluster.local:8443 \
  signet.v1.SecretsService/GetSecret \
  '{"namespace":"payments","service":"api","name":"stripe-key"}'
```

A successful response contains the decrypted `value` field.

---

## Next steps

- [installation.md](installation.md) — full production configuration reference
- [secrets-workflow.md](secrets-workflow.md) — detailed SOPS + webhook setup
- [cli.md](cli.md) — complete CLI reference
- [policies.md](policies.md) — writing and managing access policies
