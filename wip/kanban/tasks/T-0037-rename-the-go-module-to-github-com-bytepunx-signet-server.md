---
id: T-0037
type: task
nature: improvement
title: Rename the Go module to github.com/bytepunx/signet/server
status: backlog
parent: S-0009
owner: alex
created: 2026-10-27T22:25:50Z
updated: 2026-10-27T22:25:50Z
transitions: []
stream: S-0009
tags: [server, go]
touches: [server/go.mod, server/buf.gen.yaml, server/cmd/signet/bundle_cmd.go, server/cmd/signet/cliconfig_test.go, server/cmd/signet/init_cmd_test.go, server/cmd/signet/init_cmd.go, server/cmd/signet/kek_cmd.go, server/cmd/signet/master_key_cmd.go, server/cmd/signet/policy_cmd.go, server/cmd/signet/repo_cmd.go, server/cmd/signet/root.go, server/cmd/signet/seal.go, server/cmd/signet/secret_cmd_test.go, server/cmd/signet/sops_key_cmd.go, server/cmd/signet/sops_key_config.go, server/cmd/signet/status.go, server/cmd/signet/unseal.go, server/cmd/signetd/kube_unseal_test.go, server/cmd/signetd/kube_unseal.go, server/cmd/signetd/main.go, server/gen/admin/v1/admin.pb.go, server/gen/signet/v1/secrets.pb.go, server/internal/api/admin_test.go, server/internal/api/admin.go, server/internal/api/deps.go, server/internal/api/errors_test.go, server/internal/api/errors.go, server/internal/api/gitops_get_config_test.go, server/internal/api/gitops_get_sops_public_key_test.go, server/internal/api/gitops_patch_config_test.go, server/internal/api/gitops_put_config_test.go, server/internal/api/gitops_test.go, server/internal/api/gitops.go, server/internal/api/kcv.go, server/internal/api/lockmanager.go, server/internal/api/locks_test.go, server/internal/api/locks.go, server/internal/api/secrets_test.go, server/internal/api/secrets.go, server/internal/api/webhook_test.go, server/internal/api/webhook.go, server/internal/audit/audit.go, server/internal/auth/auth.go, server/internal/auth/checker_test.go, server/internal/gitops/audit_test.go, server/internal/gitops/dedup_test.go, server/internal/gitops/deps.go, server/internal/gitops/kek_test.go, server/internal/gitops/kek.go, server/internal/gitops/keys_test.go, server/internal/gitops/keys.go, server/internal/gitops/sync_test.go, server/internal/gitops/sync.go, server/internal/server/admin_tls_test.go, server/internal/server/server_test.go, server/internal/server/server.go, server/internal/unseal/manager_test.go, server/internal/unseal/manager.go, server/internal/unseal/shamir_test.go, server/internal/unseal/shamir.go]
---
# T-0037 Rename the Go module to github.com/bytepunx/signet/server

## Work

Change `server/go.mod` to `module github.com/bytepunx/signet/server` so `go install github.com/bytepunx/signet/server/cmd/signet@server/vX.Y.Z` resolves against the mono-repo tag form.

- Rewrite every import of `github.com/bytepunx/signet/` to `github.com/bytepunx/signet/server/` in the Go files listed in the touches (58 files under `server/cmd` and `server/internal`, plus the two generated files under `server/gen` that import each other).
- Change `go_package_prefix` in `server/buf.gen.yaml` to `github.com/bytepunx/signet/server/gen`.
- The generated `.pb.go` files embed `go_package` in their raw descriptors, so do not patch them with sed alone: regenerate them with `make proto` (`buf generate`, which reaches the BSR) and commit the result. If `buf generate` cannot run here, rewrite the import lines by hand and say in the commit that the descriptors still carry the old `go_package` until the next regeneration.
- Run `make build` and `make test` in `server/`, through the Docker toolchain where the host lacks Go.

This task has no `after`: it touches only `server/` Go sources and runs with the workflow tasks.

## Done when

- `server/go.mod` declares `github.com/bytepunx/signet/server` and no Go file under `server/` imports the old path.
- `server/buf.gen.yaml` has `go_package_prefix: github.com/bytepunx/signet/server/gen`.
- `make build` and `make test` pass in `server/`.
- `flai check --strict` is clean.

## Notes
