---
id: S-0010
type: story
nature: feature
title: "Proto publish workflow: BSR push from main and label on proto/v* tags"
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-10-27T19:48:21Z
transitions: []
tags: [proto]
topics: [release, github]
touches: [".github/workflows/release-proto.yml", ".github/workflows/publish-proto-main.yml", proto/README.md, proto/buf.yaml]
after: [S-0007]
---
# S-0010 Proto publish workflow: BSR push from main and label on proto/v* tags

## Goal

The schema keeps reaching the Buf Schema Registry as `buf.build/bytepunx/signet-proto`: every push to `main` that changes `proto/` runs `buf push` from `proto/`, and a push of `proto/vX.Y.Z` labels the BSR module `vX.Y.Z` and creates a GitHub Release named by the tag, as the old `proto/.github/workflows/publish.yml` and `release.yml` did.

## Acceptance criteria
- [ ] `publish-proto-main.yml` pushes to the BSR only when `proto/` changed on `main`
- [ ] `release-proto.yml` triggers on `proto/v*` tags, labels the BSR module with the version without the prefix, and creates the GitHub Release with notes from the tag annotation
- [ ] `proto/README.md` describes the mono-repo layout, the tag form, and that in-repo consumers generate from the directory while external consumers pin the BSR label

## Tasks

## Notes
