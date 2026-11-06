---
id: S-0003
type: story
nature: improvement
title: Move the Go modules to paths that resolve from this repository
status: backlog
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-11-03T18:25:54Z
transitions: []
tags: []
after: [S-0001]
---
# S-0003 Move the Go modules to paths that resolve from this repository

## Goal

`server/` is the Go module `github.com/bytepunx/signet/server` and `client-go/` is `github.com/bytepunx/signet/client-go`, tagged `server/vX.Y.Z` and `client-go/vX.Y.Z`, so `go get`, `go install`, and the module proxy work against this repository, as [ADR-0002](../../../design/adrs/0002-the-go-modules-are-github-com-bytepunx-signet-server-and-github-com-bytepunx.md) decides. Users install the CLI with `go install github.com/bytepunx/signet/server/cmd/signet@latest`. No `go.work` is committed, and the root has no `go.mod`.

## Acceptance criteria
- [ ] `server/go.mod` declares `github.com/bytepunx/signet/server`; every import in `server/` and `server/buf.gen.yaml`'s `go_package_prefix` (`github.com/bytepunx/signet/server/gen`) follow it, with `server/gen` regenerated rather than edited; `go build ./...` and `go test ./...` pass in `server/`.
- [ ] `client-go/go.mod` declares `github.com/bytepunx/signet/client-go`; imports, `buf.gen.yaml` (`go_package_prefix: github.com/bytepunx/signet/client-go/gen`), and the four examples follow it; `go build ./...` and `go test ./...` pass in `client-go/`.
- [ ] `go install github.com/bytepunx/signet/server/cmd/signet@<ref>` and `go get github.com/bytepunx/signet/client-go@<ref>` work from a clean machine against a pushed ref, and the README and docs instructions show those paths.
- [ ] No root `go.work` is committed, and the root `.gitignore` ignores `go.work` and `go.work.sum`.
- [ ] `client-go/CHANGELOG.md` and `server/CHANGELOG.md` carry a note that the module path changed, from which path to which, and that importers change their imports and `require` lines; the old paths get no further releases.
- [ ] The operator has been asked, on a thread, before a last release of `github.com/bytepunx/signet-clients/go` in bytepunx/signet-clients with a `// Deprecated: use github.com/bytepunx/signet/client-go.` comment, and the answer is carried out.

## Tasks

## Notes

Waits for S-0001: it reads ADR-0002, which S-0001 records, for the module paths and the `go.work` decision.

Current paths: `github.com/bytepunx/signet` (server, at `server/`) and `github.com/bytepunx/signet-clients/go` (client-go). The rewrite is 126 import lines in 58 files in `server/` and 12 lines in 9 files in `client-go/` (ADR-0002). The existing tags `server/v0.11.0` and `client-go/v0.7.0` do not carry the new paths; the first resolvable releases are the first `server/v*` and `client-go/v*` tags after this story, per ADR-0003. The root `v*` tags stay, so `github.com/bytepunx/signet` remains resolvable at `v0.11.0`.
