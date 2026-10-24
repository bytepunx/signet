---
id: S-0006
type: story
nature: feature
title: Release and publish the server, proto, and clients from the mono-repo
status: backlog
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-10-24T22:34:01Z
transitions: []
tags: []
after: [S-0005]
---
# S-0006 Release and publish the server, proto, and clients from the mono-repo

## Goal

Each component publishes from this repository on its own tag, following the release model S-0001 decided: the server's CLI binaries, `signetd` images, and Helm chart; the proto module to the Buf Schema Registry if kept; and the Go, Python, TypeScript, Rust, and C# packages to their registries. `install.sh` finds the right server release.

## Acceptance criteria
- [ ] The per-folder `release-please-config.json` and `.release-please-manifest.json` in `server/` and `proto/` are removed, or replaced by one root multi-component configuration if the ADR kept release-please.
- [ ] A root release workflow triggers on `server/v*` and produces what `server/.github/workflows/release.yml` did: CLI binaries for linux and darwin on amd64 and arm64 with `sha256sums.txt`, multi-arch `signetd` images on ghcr.io and Docker Hub, the Helm chart as an OCI artifact, and a GitHub release with notes from `server/CHANGELOG.md`. The `main` image publish from `publish.yml` is kept, path-filtered to `server/`.
- [ ] A root release workflow triggers on `proto/v*` and does what `proto/.github/workflows/release.yml` did, or the BSR steps are dropped per the ADR.
- [ ] One workflow per client triggers on `client-<lang>/v*` and does what the old `publish-go.yml`, `publish-python.yml`, `publish-typescript.yml`, `publish-rust.yml`, and `publish-csharp.yml` did, with the package metadata (`pyproject.toml`, `package.json`, `Cargo.toml`, `SignetClient.csproj`) naming this repository.
- [ ] `install.sh` resolves the latest `server/v*` release rather than `releases/latest`, and installs from a clean machine.
- [ ] Each component's `CHANGELOG.md` is the file the release tooling writes, and `flai release --pending` (or the chosen tool) computes the next version of each from the commits since its last tag.
- [ ] Required repository secrets (`BUF_TOKEN`, `DOCKERHUB_*`, package registry tokens, `RELEASE_PLEASE_TOKEN` if kept) are listed in `docs/contributors/` with what each is for; none is committed.
- [ ] One dry run per workflow (`workflow_dispatch` or a pre-release tag) succeeds, with run links in the story notes.

## Tasks

## Notes

Existing tags: `v0.2.0`..`v0.11.0` (old server), `server/v0.11.0`, `proto/v0.4.0`, `client-go/v0.7.0`, `client-python/v0.5.0`, `client-rust/v0.6.0`, `client-csharp/v0.6.0`, `client-typescript/v0.9.0`, and `legacy/*`. Publishing to registries reaches outside the repository: confirm each first publish with the operator.
