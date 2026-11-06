---
id: S-0006
type: story
nature: feature
title: Release and publish the server, proto, and clients from the mono-repo
status: backlog
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-11-03T18:26:05Z
transitions: []
tags: []
after: [S-0005]
---
# S-0006 Release and publish the server, proto, and clients from the mono-repo

## Goal

Each component publishes from this repository on its own `<project>/vX.Y.Z` tag, computed and pushed by `flai release --pending`, as [ADR-0003](../../../design/adrs/0003-each-component-releases-under-its-own-project-vx-y-z-tag-computed-by-flai.md) decides: the server's CLI binaries, `signetd` images, and Helm chart; the proto module's label on the Buf Schema Registry mirror ([ADR-0001](../../../design/adrs/0001-every-consumer-generates-its-stubs-from-proto-through-a-buf-directory-input-the.md)); and the Go, Python, TypeScript, Rust, and C# packages to their registries. release-please is retired, each release's changelog is its GitHub release notes, version fields are stamped from the tag, `install.sh` filters the releases for the newest `server/v*`, and the root `v*` and `legacy/*` tags stay as they are.

## Acceptance criteria
- [ ] The per-folder `release-please-config.json`, `.release-please-manifest.json`, and `release-please.yml` in `server/` and `proto/` are removed, and no release-please configuration remains.
- [ ] A root release workflow triggers on `server/v*` and produces what `server/.github/workflows/release.yml` did: CLI binaries for linux and darwin on amd64 and arm64 with `sha256sums.txt`, multi-arch `signetd` images on ghcr.io and Docker Hub, the Helm chart as an OCI artifact with `version` and `appVersion` stamped from the tag, and a GitHub release marked latest with notes from the commit subjects since the previous `server/v*` tag that change `server/`. The `main` image publish from `publish.yml` is kept, path-filtered to `server/`.
- [ ] A root workflow pushes the proto module to the BSR on changes to `proto/` on `main`, and a root release workflow triggers on `proto/v*` and labels the module with the tag's version (`proto/v0.5.0` labels `v0.5.0`).
- [ ] One workflow per client triggers on `client-<lang>/v*` and does what the old `publish-go.yml`, `publish-python.yml`, `publish-typescript.yml`, `publish-rust.yml`, and `publish-csharp.yml` did, stamping the version in `pyproject.toml`, `package.json`, `Cargo.toml`, or `SignetClient.csproj` from the tag in its own checkout, with the package metadata naming this repository and the GitHub release passing `make_latest: false`.
- [ ] `install.sh` lists the releases and installs the newest non-draft, non-prerelease one whose tag starts with `server/` (or the one `SIGNET_VERSION` names), a download URL whose tag holds a slash is confirmed to resolve (URL-encoded if needed), and it installs from a clean machine.
- [ ] Each component's `CHANGELOG.md` carries one last hand-written entry for the move to this repository and points at the GitHub releases; `flai release --pending` computes the next version of each from what was accepted since its last tag; and `design/conventions/git.md` gains a project addition saying a component's changelog is its GitHub release notes and its version fields are stamped from the tag.
- [ ] No workflow triggers on the root `v*` or `legacy/*` tags, and none of them is moved or deleted.
- [ ] Required repository secrets (`BUF_TOKEN`, `DOCKERHUB_*`, package registry tokens) are listed in `docs/contributors/` with what each is for; none is committed.
- [ ] One dry run per workflow (`workflow_dispatch` or a pre-release tag) succeeds, with run links in the story notes.

## Tasks

## Notes

Waits for S-0005: its release workflows sit beside the root CI workflows and path filters S-0005 builds (ADR-0003).

Existing tags: `v0.2.0`..`v0.11.0` (old server), `server/v0.11.0`, `proto/v0.4.0`, `client-go/v0.7.0`, `client-python/v0.5.0`, `client-rust/v0.6.0`, `client-csharp/v0.6.0`, `client-typescript/v0.9.0`, and `legacy/*`. The existing `<project>/v*` tags are each component's last release (ADR-0003). Publishing to registries reaches outside the repository: confirm each first publish with the operator.
