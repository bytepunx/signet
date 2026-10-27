---
id: T-0047
type: task
nature: feature
title: Go client test script in Docker
status: backlog
parent: S-0007
owner: alex
created: 2026-10-27T22:26:21Z
updated: 2026-10-27T22:26:21Z
transitions: []
stream: S-0007
tags: [client-go, ci]
touches: [scripts/test-client-go.sh]
after: [T-0042]
---
# T-0047 Go client test script in Docker

## Work

Add `scripts/test-client-go.sh`, POSIX `sh` under `set -eu`, modelled on `scripts/test-client-elixir.sh`: with `client-go/` mounted in the `golang` image matching `client-go/go.mod` (1.26), run `go vet ./...` and `go test -race -count=1 ./...`. Before that, run `scripts/buf-generate.sh client-go` and fail if `git diff --quiet client-go/gen` finds drift, as the old signet-clients CI did for committed stubs. Extra arguments run instead of the default test command, as the Elixir script does.

It waits for `scripts/buf-generate.sh` (T-0042).

## Done when

- `scripts/test-client-go.sh` exits 0 on a clean checkout and runs the unit tests (`client_test.go`, `restart_test.go`, `sops_encrypt_test.go`).
- Editing a file under `client-go/gen/` makes it fail with the diff.
- No tool but Docker is needed on the host, and `sh -n` is clean.

## Notes

Layer 2 of the plan. The module path in `client-go/go.mod` still names the old repository; story S-0011 changes it, and this script does not depend on the path.
