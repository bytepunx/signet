# signet

[![CI](https://github.com/bytepunx/signet/actions/workflows/ci.yml/badge.svg)](https://github.com/bytepunx/signet/actions/workflows/ci.yml)

**Configuration and secrets management for Kubernetes, built on SPIFFE/SPIRE workload identity.**

signet gives every workload in your cluster a secure, zero-long-lived-credential way to fetch
the secrets and configuration it needs — authenticated by *what the workload is* (its SPIFFE
identity), not by a token it has to be handed, rotate, and protect. Secrets live in git,
encrypted with [SOPS](https://github.com/getsops/sops) and [age](https://github.com/FiloSottile/age);
signet decrypts them once, on the server side, and re-encrypts each one under its own key before
it ever touches disk.

---

## Why signet

**Kubernetes Secrets** are base64, not encryption. They're `list`-able at the namespace level
with no field-level access control, replicate to every node that schedules a pod using them, and
have no built-in rotation, versioning, or audit trail. signet stores secrets encrypted with a
per-secret data key, wrapped by a key-encryption key, wrapped by a master key that never touches
disk — and every access, permitted or denied, is written to a tamper-evident audit log.

**HashiCorp Vault** is a general-purpose secrets platform: dynamic secrets, PKI issuance, cloud
IAM brokering, a matrix of auth methods and secrets engines. That breadth is real operational
surface — policies to write for every path, an auth method to configure for every consumer, a
storage backend and (for HA) a separate Enterprise tier to reason about. signet does one thing:
get the right secret to the right Kubernetes workload. A workload requesting its *own* namespace
and service account's secrets needs **zero configuration** — access is granted by the
exact-match convention on its SPIFFE identity. Cross-service access is the only case that needs
an explicit policy.

**External Secrets Operator / Sealed Secrets** solve *ingestion into* Kubernetes Secrets, which
means the underlying weaknesses of Kubernetes Secrets (above) are still the last mile. signet
replaces that last mile entirely: workloads talk to signet directly over authenticated mTLS, and
plaintext never lands in a Kubernetes Secret object.

**Cloud KMS + Secrets Manager** (AWS/GCP/Azure) work well if you're single-cloud and don't mind
the coupling. signet has no cloud dependency: age keys are generated and held in-cluster, and
Shamir's Secret Sharing, a direct operator-held key, or (with an accepted trust-boundary
tradeoff) a Kubernetes Secret all work as the unseal mechanism — including fully air-gapped.

**Bare SOPS-in-git** gives you encryption at rest and PR review for free, but no distribution
story: something still has to decrypt those files and get the values to running workloads
without leaking them into environment variables, `/proc`, or crash dumps. signet *is* that
something — it's the server-side half of a SOPS+age workflow you may already be using.

In short: if your environment is already Kubernetes + SPIFFE/SPIRE, or you want per-secret
envelope encryption, git-native secret provenance, and workload-identity-based access control
without adopting a much larger platform, signet is a narrowly-scoped fit. If you need dynamic
database credentials, PKI issuance, or multi-cloud secret brokering, Vault's breadth will serve
you better.

## Key features

- **SPIFFE/SPIRE workload identity** — mTLS authentication using short-lived SVIDs; no
  long-lived credential to leak or rotate.
- **Convention-first authorization** — a workload's own namespace/service secrets are accessible
  with no policy configuration; [explicit policies](docs/policies.md) handle everything else.
- **Envelope encryption** — DEK → KEK → master key hierarchy, AES-256-GCM with AAD binding so a
  ciphertext can't be silently swapped between rows by a party with database access but no keys.
- **Git as source of truth** — secrets are SOPS/age-encrypted in a git repo; signet syncs on
  webhook push and via periodic reconciliation. Full history, PR review, branch-based promotion.
- **No forced cloud/KMS dependency** — Shamir's Secret Sharing, a direct operator key, or a
  Kubernetes Secret (with a documented trust tradeoff) unseal the master key; age keys are
  generated and stored by signet itself.
- **Live updates** — workloads can watch a secret, a config document, or a whole service bundle
  and react to changes via gRPC streaming, instead of polling.
- **Coordinated rolling restarts** — a built-in [distributed lock](docs/restart-lock.md) lets
  replicas restart one at a time when a bundle changes, instead of a thundering herd.
- **Tamper-evident audit log** — every secret access, permitted or denied, is HMAC-chained and
  forwarded out-of-cluster; fail-closed by default if the audit write itself fails.

## Quickstart

```bash
helm install signet oci://ghcr.io/bytepunx/charts/signet \
  --namespace signet --create-namespace \
  --set signet.trustDomain=<your-trust-domain> \
  --set signet.existingSecret=signet-secrets
```

New to signet? Start with **[Getting Started](docs/getting-started.md)** — a ~30 minute walkthrough
from a bare cluster to a workload successfully retrieving its first secret.

## Documentation

| Document | Covers |
|---|---|
| [Getting Started](docs/getting-started.md) | Zero-to-first-secret walkthrough: deploy SPIRE, install signet, unseal, register a repo |
| [Installation](docs/installation.md) | Prerequisites, Helm chart reference, every configuration value |
| [CLI Reference](docs/cli.md) | Operator subcommands: `init`, `unseal`, `seal`, `status`, `sops-key`, `repo`, `config` |
| [Secrets Workflow](docs/secrets-workflow.md) | End-to-end SOPS/age encryption workflow, from generating a key to a workload reading the value |
| [Configuration Values](docs/configuration.md) | Distributing non-secret config (feature flags, endpoints) alongside encrypted secrets |
| [Access Policies](docs/policies.md) | The convention-first authorization model and how to write explicit cross-service policies |
| [Coordinated Rolling Restarts](docs/restart-lock.md) | The distributed restart-lock primitive and why it exists |

### Design & security

| Document | Covers |
|---|---|
| [Architecture Decision Summary](design/draft.md) | Every major design decision — storage, envelope encryption, unsealing, PKI, GitOps sync — with rationale and alternatives considered |
| [Security Findings & Hardening Checklist](design/security-findings.md) | A self-audit of the codebase against its own design, with severities, fixes, and test coverage tracked to completion |

signet's design documentation is kept in sync with the implementation as a project convention —
if you're evaluating signet for a security-sensitive deployment, the findings document above is
a good look at how seriously that's taken in practice, not just what's claimed.

## Project status

signet is early-stage (`v0.2.x`) and under active development. The core secret-serving,
GitOps sync, unsealing, and authorization paths are implemented and tested; see
[CHANGELOG.md](CHANGELOG.md) for release history.
