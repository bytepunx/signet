---
id: S-0011
type: story
nature: feature
title: "Go client: module path for the mono-repo and release workflow on client-go/v* tags"
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-10-27T19:48:21Z
transitions: []
tags: [client-go]
topics: [release, github]
touches: [".github/workflows/release-client-go.yml", client-go/go.mod, client-go/client.go, client-go/restart.go, client-go/sops_encrypt.go, client-go/examples, client-go/README.md, client-go/buf.gen.yaml]
after: [S-0007]
---
# S-0011 Go client: module path for the mono-repo and release workflow on client-go/v* tags

## Goal

The Go client is `github.com/bytepunx/signet/client-go`, resolvable with `go get github.com/bytepunx/signet/client-go@vX.Y.Z` because flai's `client-go/vX.Y.Z` tag is exactly the tag form Go requires for a module in a subdirectory. `release-client-go.yml` runs on that tag: regenerates stubs, vets, tests, asks `proxy.golang.org` to index the version, and creates the GitHub Release. The README's install and publish sections describe the new path and tags.

## Acceptance criteria
- [ ] `client-go/go.mod` is `github.com/bytepunx/signet/client-go`, imports and the `go_package_prefix` follow, and `go test ./...` passes
- [ ] `release-client-go.yml` triggers on `client-go/v*` only, verifies, indexes the module on the proxy, and creates the GitHub Release with notes from the tag annotation
- [ ] `client-go/README.md` names the new module path, the tag form, and the workflow

## Tasks

## Notes

Reference: signet-clients' `publish-go.yml` on GitHub. The module path change is breaking for importers of the old path.
