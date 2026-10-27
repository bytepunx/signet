---
id: T-0056
type: task
nature: improvement
title: Rename the Go module to github.com/bytepunx/signet/client-go and regenerate the stubs
status: backlog
parent: S-0011
owner: alex
created: 2026-10-27T22:51:51Z
updated: 2026-10-27T22:51:51Z
transitions: []
stream: S-0011
tags: [client-go]
touches: [client-go/go.mod, client-go/buf.gen.yaml, client-go/client.go, client-go/restart.go, client-go/restart_test.go, client-go/sops_encrypt.go, client-go/examples/echo/main.go, client-go/examples/getsecret/main.go, client-go/examples/gitops-workload/main.go, client-go/examples/restart-on-change/main.go, client-go/gen/admin/v1/admin.pb.go, client-go/gen/admin/v1/admin_grpc.pb.go, client-go/gen/signet/v1/secrets.pb.go, client-go/gen/signet/v1/secrets_grpc.pb.go]
---
# T-0056 Rename the Go module to github.com/bytepunx/signet/client-go and regenerate the stubs

## Work

Change the module line of `client-go/go.mod` to `github.com/bytepunx/signet/client-go`. Change `go_package_prefix` in `client-go/buf.gen.yaml` to `github.com/bytepunx/signet/client-go/gen`. Rewrite every import of `github.com/bytepunx/signet-clients/go` to the new path in `client.go`, `restart.go`, `restart_test.go`, and the four `examples/*/main.go`. In `sops_encrypt.go`, update the comment that names the old repository's languages to this repository. Leave references to old issue numbers (`bytepunx/signet-clients#NN`) as they are: they name history, not the module.

Regenerate `client-go/gen/` with the pinned `buf.gen.yaml`, with the input unchanged (the BSR pin stays: S-0020 moves it to the `proto/proto` directory later). The stubs embed the Go package path in their raw descriptors, so they must be regenerated, not edited by hand. The `inputs` pin and plugin versions stay as they are. Run `go mod tidy` and commit `go.sum` only if it changes.

This is one task because the build is broken between the module line and the imports, and the descriptors must agree with the prefix.

## Done when

- `client-go/go.mod` declares `module github.com/bytepunx/signet/client-go`.
- No `.go` file under `client-go/` imports `github.com/bytepunx/signet-clients/go`, and `grep -rn 'signet-clients/go' client-go` finds nothing outside `CHANGELOG.md`.
- `buf generate` in `client-go/` leaves `gen/` with no diff, with the new `go_package_prefix`.
- `go vet ./...` and `go test ./...` pass in `client-go/`, and each example builds with `go build ./examples/...`.

## Notes
