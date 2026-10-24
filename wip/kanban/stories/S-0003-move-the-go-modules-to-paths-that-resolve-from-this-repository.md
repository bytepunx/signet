---
id: S-0003
type: story
nature: improvement
title: Move the Go modules to paths that resolve from this repository
status: backlog
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-10-24T22:33:36Z
transitions: []
tags: []
after: [S-0001]
---
# S-0003 Move the Go modules to paths that resolve from this repository

## Goal

`server/` and `client-go/` are Go modules whose import paths match their location under `github.com/bytepunx/signet`, so `go get`, `go install`, and the module proxy work against this repository, as S-0001 decided.

## Acceptance criteria
- [ ] `server/go.mod` declares the module path the ADR chose; every import in `server/` and `server/buf.gen.yaml`'s `go_package_prefix` follow it; `go build ./...` and `go test ./...` pass in `server/`.
- [ ] `client-go/go.mod` declares the module path the ADR chose; imports, `buf.gen.yaml`, and the four examples follow it; `go build ./...` and `go test ./...` pass in `client-go/`.
- [ ] `go install <path>/cmd/signet@<ref>` and `go get <client-go path>@<ref>` work from a clean machine against a pushed ref, and the `install.sh`, README, and docs instructions show those paths.
- [ ] A root `go.work` exists if and only if the ADR calls for one, and it is covered by `.gitignore` or committed as the ADR says.
- [ ] `client-go/CHANGELOG.md` and `server/CHANGELOG.md` carry a note that the module path changed and what consumers must do.

## Tasks

## Notes

Current paths: `github.com/bytepunx/signet` (server, at `server/`) and `github.com/bytepunx/signet-clients/go` (client-go). Go tags for a module in a subdirectory must be `<dir>/vX.Y.Z`; `client-go/v0.7.0` and `server/v0.11.0` already exist.
