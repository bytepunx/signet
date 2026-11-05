---
id: D-0020
title: "Plan for S-0009: six tasks in three layers, and the cost of delay inputs: accept"
kind: answer
sources: [TH-0006]
topics: [release, github, ci, server, go, code, all]
paths: [".github/workflows/release-server.yml", ".github/workflows/publish-server-main.yml", server/go.mod, server/buf.gen.yaml, server/gen, server/Makefile, server/deploy/helm/signet/Chart.yaml, server/docs/installation.md, server/README.md, install.sh, server/cmd/signet/bundle_cmd.go, server/cmd/signet/cliconfig_test.go, server/cmd/signet/init_cmd.go, server/cmd/signet/init_cmd_test.go, server/cmd/signet/kek_cmd.go, server/cmd/signet/master_key_cmd.go, server/cmd/signet/policy_cmd.go, server/cmd/signet/repo_cmd.go, server/cmd/signet/root.go, server/cmd/signet/seal.go, server/cmd/signet/secret_cmd_test.go, server/cmd/signet/sops_key_cmd.go, server/cmd/signet/sops_key_config.go, server/cmd/signet/status.go, server/cmd/signet/unseal.go, server/cmd/signetd/kube_unseal.go, server/cmd/signetd/kube_unseal_test.go, server/cmd/signetd/main.go, server/internal/api/admin.go, server/internal/api/admin_test.go, server/internal/api/deps.go, server/internal/api/errors.go, server/internal/api/errors_test.go, server/internal/api/gitops.go, server/internal/api/gitops_get_config_test.go, server/internal/api/gitops_get_sops_public_key_test.go, server/internal/api/gitops_patch_config_test.go, server/internal/api/gitops_put_config_test.go, server/internal/api/gitops_test.go, server/internal/api/kcv.go, server/internal/api/lockmanager.go, server/internal/api/locks.go, server/internal/api/locks_test.go, server/internal/api/secrets.go, server/internal/api/secrets_test.go, server/internal/api/webhook.go, server/internal/api/webhook_test.go, server/internal/audit/audit.go, server/internal/auth/auth.go, server/internal/auth/checker_test.go, server/internal/gitops/audit_test.go, server/internal/gitops/dedup_test.go, server/internal/gitops/deps.go, server/internal/gitops/kek.go, server/internal/gitops/kek_test.go, server/internal/gitops/keys.go, server/internal/gitops/keys_test.go, server/internal/gitops/sync.go, server/internal/gitops/sync_test.go, server/internal/server/admin_tls_test.go, server/internal/server/server.go, server/internal/server/server_test.go, server/internal/unseal/manager.go, server/internal/unseal/manager_test.go, server/internal/unseal/shamir.go, server/internal/unseal/shamir_test.go]
date: 2026-10-29
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0020 Plan for S-0009: six tasks in three layers, and the cost of delay inputs: accept

The operator, alex, answered TH-0006 at 2026-10-29T17:54:10Z.

What was asked:

```text
Question first: S-0009 has no cost of delay inputs, and neither has E-0002, so `flai cod S-0009` refuses. I recommend time_lost_per_cycle 2h, with no revenue and no penalty: until this ships, every server release is cut by hand and the old v* tags cannot be reused. Alternatives: a penalty_per_week if a customer waits on the module path, or 0 if the release can wait for S-0008. Set the inputs with `flai edit S-0009 --time-lost-per-cycle 2h` (or on E-0002 to cover all its stories) and I will work out the value on the next run.
```

The answer:

```text
accept
```

Source: TH-0006.
