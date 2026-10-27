---
id: S-0009
type: story
nature: feature
title: "Server release workflow on server/v* tags"
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:49Z
updated: 2026-10-27T19:48:20Z
transitions: []
tags: [server]
topics: [release, github]
touches: [".github/workflows/release-server.yml", ".github/workflows/publish-server-main.yml", server/go.mod, server/buf.gen.yaml, server/gen, server/cmd, server/internal, server/Makefile, server/deploy/helm/signet/Chart.yaml, server/docs/installation.md, server/README.md, install.sh]
after: [S-0007]
---
# S-0009 Server release workflow on server/v* tags

## Goal

A push of `server/vX.Y.Z` builds and publishes everything the old `server/.github/workflows/release.yml` did, with the prefix stripped: CLI binaries for linux and darwin on amd64 and arm64 with checksums, `ghcr.io/bytepunx/signetd` and `docker.io/bytepunx/signet` images tagged `vX.Y.Z`, `vX.Y`, and `latest`, the Helm chart pushed to `oci://ghcr.io/bytepunx/charts` with version `X.Y.Z` and appVersion `vX.Y.Z`, and a GitHub Release named `server/vX.Y.Z` with the binaries attached and notes from the tag annotation and the commits since the previous `server/` tag. A push to `main` that touches `server/` publishes the `main` image tag. The Go module becomes `github.com/bytepunx/signet/server` so `go install github.com/bytepunx/signet/server/cmd/signet@server/vX.Y.Z` works, and `install.sh` resolves the newest `server/v*` release instead of `releases/latest`.

## Acceptance criteria
- [ ] `release-server.yml` triggers on `server/v*` tags only and produces the binaries, images, chart, and GitHub Release described above
- [ ] `publish-server-main.yml` pushes the `main` image only when `server/` changed
- [ ] `server/go.mod` is `github.com/bytepunx/signet/server`, every import and the `go_package_prefix` follow, and `make build` and `make test` pass in `server/`
- [ ] `install.sh` installs the newest `server/v*` release on linux and darwin and ignores other components' releases
- [ ] `server/docs/installation.md` and `server/README.md` name the new tag form, image tags, and chart version
- [ ] A test tag on a branch (for example `server/v0.0.0-test`) run of the workflow is green, then deleted

## Tasks

## Notes

A breaking change for anyone importing the old module path; this story is the one that delivers it to `server`.
