---
id: D-0021
title: "T-0045: push a test tag that publishes real images? Close-out blocked by BSR limit: Docker is already logged into in this local environment and shouldn't need separate variables"
kind: answer
sources: [TH-0017]
topics: [release, github, ci, server, go, code, all]
paths: [".github/workflows/release-server.yml", ".github/workflows/publish-server-main.yml", server/go.mod, server/buf.gen.yaml, server/gen, server/Makefile, server/deploy/helm/signet/Chart.yaml, server/docs/installation.md, server/README.md, install.sh, server/cmd/signet/bundle_cmd.go, server/cmd/signet/cliconfig_test.go, server/cmd/signet/init_cmd.go, server/cmd/signet/init_cmd_test.go, server/cmd/signet/kek_cmd.go, server/cmd/signet/master_key_cmd.go, server/cmd/signet/policy_cmd.go, server/cmd/signet/repo_cmd.go, server/cmd/signet/root.go, server/cmd/signet/seal.go, server/cmd/signet/secret_cmd_test.go, server/cmd/signet/sops_key_cmd.go, server/cmd/signet/sops_key_config.go, server/cmd/signet/status.go, server/cmd/signet/unseal.go, server/cmd/signetd/kube_unseal.go, server/cmd/signetd/kube_unseal_test.go, server/cmd/signetd/main.go, server/internal/api/admin.go, server/internal/api/admin_test.go, server/internal/api/deps.go, server/internal/api/errors.go, server/internal/api/errors_test.go, server/internal/api/gitops.go, server/internal/api/gitops_get_config_test.go, server/internal/api/gitops_get_sops_public_key_test.go, server/internal/api/gitops_patch_config_test.go, server/internal/api/gitops_put_config_test.go, server/internal/api/gitops_test.go, server/internal/api/kcv.go, server/internal/api/lockmanager.go, server/internal/api/locks.go, server/internal/api/locks_test.go, server/internal/api/secrets.go, server/internal/api/secrets_test.go, server/internal/api/webhook.go, server/internal/api/webhook_test.go, server/internal/audit/audit.go, server/internal/auth/auth.go, server/internal/auth/checker_test.go, server/internal/gitops/audit_test.go, server/internal/gitops/dedup_test.go, server/internal/gitops/deps.go, server/internal/gitops/kek.go, server/internal/gitops/kek_test.go, server/internal/gitops/keys.go, server/internal/gitops/keys_test.go, server/internal/gitops/sync.go, server/internal/gitops/sync_test.go, server/internal/server/admin_tls_test.go, server/internal/server/server.go, server/internal/server/server_test.go, server/internal/unseal/manager.go, server/internal/unseal/manager_test.go, server/internal/unseal/shamir.go, server/internal/unseal/shamir_test.go]
date: 2026-10-29
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0021 T-0045: push a test tag that publishes real images? Close-out blocked by BSR limit: Docker is already logged into in this local environment and shouldn't need separate variables

The operator, alex, answered TH-0017 at 2026-10-29T17:59:11Z, on story S-0009's T-0045, which pushes a test tag (`server/v0.0.0-test`) that would publish real images to ghcr.io and Docker Hub.

What was asked:

```text
T-0037, T-0039, T-0040, T-0041 and T-0043 are done on story/S-0009. Two things need you.
```

The answer:

```text
Docker is already logged into in this local environment and shouldn't need separate variables. The secret for the org that is set on signet is BSR_AUTH_TOKEN, not BUF_TOKEN
```

The fact for later stories: the BSR credential set on the repository is the secret `BSR_AUTH_TOKEN`, not `BUF_TOKEN`, and Docker Hub login is already present in the operator's local environment, so a story need not wait for separate Docker variables. `design/system/release.md` still names `BUF_TOKEN` for the proto workflows, which disagrees with this answer, so the entry stays active.

Source: TH-0017.
