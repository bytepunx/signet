---
id: T-0046
type: task
nature: feature
title: "Server test script: go test, golangci-lint, make build, and stub drift in Docker"
status: backlog
parent: S-0007
owner: alex
created: 2026-10-27T22:26:18Z
updated: 2026-10-27T22:27:03Z
transitions: []
stream: S-0007
tags: [server, ci]
touches: [scripts/test-server.sh]
after: [T-0042]
---
# T-0046 Server test script: go test, golangci-lint, make build, and stub drift in Docker

## Work

Add `scripts/test-server.sh`, POSIX `sh` under `set -eu`, that runs the server's old CI jobs (`server/.github/workflows/ci.yml`: Test, Lint, Proto generated files, Build binaries) inside Docker, with `server/` mounted and a Go module and build cache volume kept between runs:

1. `go test -race -count=1 ./...` (the `make test` target) in the `golang` image matching `server/go.mod` (1.26). `-race` needs CGO, so use the Debian image, not alpine.
2. `golangci-lint run ./...` in a pinned `golangci/golangci-lint` image. `server/.golangci.yml` is the v1 schema and `server/githooks/pre-commit` pins v1.64.8, so pin the v1 image (v1.64.8) rather than `latest`, which the old job used and which would read the file as v2; record the pin in the script.
3. `make build` in the Go image.
4. Stub drift: run `scripts/buf-generate.sh server`, then fail if `git diff --quiet server/gen` finds a difference, printing the diff and telling the developer to run `buf generate` and commit. Unlike the old job, it never commits a fix back to the pull request branch.

It waits for `scripts/buf-generate.sh` (T-0042), which step 4 calls. The Helm checks are a separate task that this script then calls.

## Done when

- `scripts/test-server.sh` exits 0 on a clean checkout and runs the four steps in order, printing each.
- Editing a file under `server/gen/` makes step 4 fail with the diff.
- A deliberate lint error in a scratch copy makes step 2 fail.
- No tool but Docker is needed on the host, and `sh -n` is clean.

## Notes

Layer 2 of the plan. Record the image versions in `design/tech/` only for a tool it does not list yet. The old `docker-build` job (build the `signetd` and `signet` images without pushing) is left to story S-0009, which builds those images on release.
