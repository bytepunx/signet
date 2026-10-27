---
id: S-0006
type: story
nature: improvement
title: Baseline component tags and retire release-please
status: ready
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:49Z
updated: 2026-10-27T22:23:53Z
transitions:
  - to: ready
    at: 2026-10-27T22:23:53Z
    by: alex
tags: [server]
topics: [release]
touches: [server/.github/workflows/ci.yml, server/.github/workflows/publish.yml, server/.github/workflows/release-please.yml, server/.github/workflows/release.yml, server/.github/FUNDING.yml, server/release-please-config.json, server/.release-please-manifest.json, proto/.github/workflows/ci.yml, proto/.github/workflows/publish.yml, proto/.github/workflows/release-please.yml, proto/.github/workflows/release.yml, proto/release-please-config.json, proto/.release-please-manifest.json, client-erlang/README.md, system-flow.yaml, client-elixir/README.md]
---
# S-0006 Baseline component tags and retire release-please

## Goal

flai continues each component's version numbering from where the old repositories left off, and nothing of release-please remains: the dead `.github/` folders and release-please files under `server/` and `proto/` are deleted, `client-erlang/` stops being a component (its toolchain note moves into the Elixir client README and the folder and its `system-flow.yaml` entry go), and annotated baseline tags are pushed on the import commits: `server/v0.11.0`, `proto/v0.4.0`, `client-go/v0.7.0`, `client-python/v0.5.0`, `client-rust/v0.6.0`, `client-typescript/v0.9.0`, `client-csharp/v0.6.0`.

## Acceptance criteria
- [ ] `flai release --pending --dry-run` plans no component from `0.0.0` other than `client-elixir`, and plans nothing for `client-erlang`
- [ ] The baseline tags exist on the remote on the commits that imported each component, with annotations naming the old repository's tag they continue
- [ ] No `release-please` file, manifest, or workflow remains anywhere in the repository, and `server/.github/` and `proto/.github/` are gone
- [ ] `client-erlang/` is gone, `system-flow.yaml` no longer lists it, and the Elixir client README keeps the note on why Buf's registry cannot generate BEAM stubs

## Tasks

## Notes

Pushing tags is the operator's action (safety.md); the story prepares the exact commands in the narrative and the runbook. The old root `v*` tags stay as history of the server. Publishing this story releases a patch of `server` and `proto` (deleted files only), which is the first exercise of the new workflows once they exist.
