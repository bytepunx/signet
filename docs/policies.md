# Access policies

signet uses SPIFFE/SPIRE workload identity to control which services can
retrieve which secrets. The design follows a **convention-first** model: the
most common case requires no configuration, and explicit policies only become
necessary when access crosses the boundaries of a workload's own identity.

---

## The exact-match convention (no policy required)

When a workload's SPIFFE ID encodes a Kubernetes identity of the form:

```
spiffe://<trust-domain>/ns/<namespace>/sa/<service-account>
```

and the `<namespace>` and `<service-account>` exactly match the secret's
`namespace` and `service` fields, access is granted automatically — no policy
needs to be created.

This is the primary intended usage pattern: a service reads the secrets that
belong to it, as established by the path convention operators use when
encrypting files with SOPS.

**Example:** the `api` workload in the `payments` Kubernetes namespace presents
the SVID `spiffe://cluster.local/ns/payments/sa/api`. It can fetch any secret
stored under `(namespace=payments, service=api)` — such as
`payments/api/stripe-key` or `payments/api/db-password` — without any
operator action beyond ensuring the SOPS files were placed under the correct
path in the repository.

---

## How policies work

When the exact-match convention does not apply, signet consults the policy
store. A policy is a row that pairs a SPIFFE ID pattern with a secret scope:

1. signet reads the workload's X.509 SVID from the mTLS handshake
2. It extracts the SPIFFE ID (e.g. `spiffe://cluster.local/ns/payments/sa/api`)
3. It checks the exact-match convention (above)
4. If the convention does not apply, it checks whether any stored policy
   matches both the SPIFFE ID and the requested secret's namespace and service
5. If at least one policy matches, the secret is returned. Otherwise,
   `PERMISSION_DENIED`

Policies are required for every access pattern that crosses the workload's own
namespace/service boundary: shared secrets, cross-namespace access, wildcard
grants, and so on.

---

## SPIFFE ID matching in policies

Policy SPIFFE ID fields support glob wildcards, evaluated with Go's
[`path.Match`](https://pkg.go.dev/path#Match): `*` matches any sequence of
characters **within a single `/`-delimited segment** — it does not cross
`/`. There is no `**` (cross-segment) wildcard. Because every SPIFFE ID here
follows the fixed `ns/<namespace>/sa/<service-account>` shape, a single `*`
per segment already covers every case that matters — "any workload in the
trust domain" is `ns/*/sa/*`, not a bare `**`.

| Pattern | Matches | Does not match |
|---|---|---|
| `spiffe://cluster.local/ns/payments/sa/api` | Exact: `payments/api` SA only | Any other SA |
| `spiffe://cluster.local/ns/payments/sa/*` | Any SA in `payments` namespace | SAs in other namespaces |
| `spiffe://cluster.local/ns/*/sa/prometheus` | `prometheus` SA in any namespace | Other SA names |
| `spiffe://cluster.local/ns/*/sa/*` | Any SPIFFE ID in the trust domain | IDs from other trust domains |

Wildcards are useful for monitoring SAs and shared secrets but should be
used conservatively for sensitive secrets.

---

## Creating policies

Policies are needed only when a workload requires access **outside its own
namespace/service**. They are managed via the admin gRPC API:

```bash
TOKEN=$(kubectl create token signet-admin -n signet --duration=1h)
```

### Grant access to a shared secret

Allow the `payments/api` workload to also read a shared database secret stored
under `infra/db`:

```bash
signet policy create \
  --spiffe-id "spiffe://cluster.local/ns/payments/sa/api" \
  --namespace  infra \
  --service    db \
  --token      "$TOKEN"
```

### Grant a namespace-wide service access to shared secrets

Allow any service in the `infra` namespace to read its own shared secrets:

```bash
signet policy create \
  --spiffe-id "spiffe://cluster.local/ns/infra/sa/*" \
  --namespace  infra \
  --service    shared \
  --token      "$TOKEN"
```

### Cross-namespace access

Allow a monitoring workload in `observability` to read a specific secret from
`payments`:

```bash
signet policy create \
  --spiffe-id "spiffe://cluster.local/ns/observability/sa/metrics-collector" \
  --namespace  payments \
  --service    api \
  --token      "$TOKEN"
```

### Wildcard trust-domain grant (dev/test only)

Grant all workloads in the trust domain access to a shared dev secret. Use
only in non-production environments:

```bash
signet policy create \
  --spiffe-id "spiffe://cluster.local/ns/*/sa/*" \
  --namespace  dev \
  --service    shared \
  --token      "$TOKEN"
```

---

## Listing policies

```bash
signet policy list --token "$TOKEN"
```

**Example output:**
```
ID           SPIFFE ID                                     NAMESPACE    PATTERN                   PERMISSIONS
policy-1     spiffe://cluster.local/ns/payments/sa/api      infra        infra/db/*                get
policy-2     spiffe://cluster.local/ns/infra/sa/*           infra        infra/shared/*            get
policy-3     spiffe://cluster.local/ns/observability/sa/... payments     payments/api/*            get
```

`PATTERN` is the full `namespace/service/secret_name` glob as stored — `--service` and an
optional `--secret-name` (default `*`, meaning every secret in that namespace/service) are
combined into it at creation time; they aren't tracked as separate columns.

---

## Removing policies

```bash
signet policy remove --id a1b2c3d4-... --token "$TOKEN"
```

Removing a policy takes effect immediately. Workloads that held access only
through the removed policy will receive `PERMISSION_DENIED` on their next
`GetSecret` call.

The exact-match convention is not a stored policy and cannot be removed. If a
workload should not be able to read its own secrets, the SOPS files should
simply not be created for that namespace/service path.

---

## Auditing

Every `GetSecret` call — whether allowed or denied, and whether permitted by
the exact-match convention or by a stored policy — is recorded in the audit
log with:

- Timestamp
- SPIFFE ID of the caller
- Requested namespace, service, and secret name
- Whether it was allowed or denied
- HMAC chain entry (tamper-evident)

```bash
signet audit list --namespace payments --limit 50 --token "$TOKEN"
```

The audit chain key is set at deploy time via `SIGNET_AUDIT_CHAIN_KEY`. Each
entry's HMAC is chained to the previous entry; a break in the chain indicates
tampering. Audit entries are never deleted.

---

## Policy design patterns

### Principle of least privilege

The exact-match convention already embodies least privilege for the common
case. For explicit policies, grant access only to what the workload actually
needs:

```bash
# Allow a sidecar to read a shared TLS cert without broad namespace access
signet policy create \
  --spiffe-id "spiffe://cluster.local/ns/payments/sa/envoy" \
  --namespace  infra \
  --service    tls \
  --token      "$TOKEN"
```

### Shared secrets

For secrets shared across multiple services (e.g. a shared database password),
create one policy per consumer rather than using a wildcard:

```bash
for SA in api worker scheduler; do
  signet policy create \
    --spiffe-id "spiffe://cluster.local/ns/payments/sa/${SA}" \
    --namespace infra --service db \
    --token "$TOKEN"
done
```

### Short-lived operator access

For one-off operator tasks (debugging, migrations), create a policy tied to a
specific operator pod SVID, perform the task, then remove the policy:

```bash
# Grant (before the task)
signet policy create \
  --spiffe-id "spiffe://cluster.local/ns/ops/sa/migration-runner" \
  --namespace payments --service api \
  --token "$TOKEN"

# ... perform migration ...

# Revoke
signet policy remove --id <policy-id> --token "$TOKEN"
```
