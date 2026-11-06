---
id: S-0010
type: story
nature: feature
title: "Proto publish workflow: BSR push from main and label on proto/v* tags"
status: done
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-11-06T07:42:53Z
transitions:
  - to: ready
    at: 2026-10-27T22:54:48Z
    by: alex
  - to: in-progress
    at: 2026-10-29T18:02:29Z
    by: agent-S-0010
  - to: review
    at: 2026-11-03T02:48:54Z
    by: agent-S-0010
  - to: done
    at: 2026-11-05T19:24:45Z
    by: alex
blocked:
  - from: 2026-10-29T18:24:30Z
    until: 2026-10-30T01:55:55Z
    reason: "Close-out test tier fails on BSR rate limit (I-0007); needs BUF_TOKEN in the environment or the limit to clear"
  - from: 2026-10-30T02:07:40Z
    until: 2026-11-06T07:42:53Z
    reason: "Close-out test tier fails on BSR rate limit (I-0007) again after retry; rerun when it clears or BUF_TOKEN is set"
tags: [proto]
topics: [release, github]
touches: [".github/workflows/release-proto.yml", ".github/workflows/publish-proto-main.yml", proto/README.md, proto/buf.yaml, design/issues/I-0007-the-bsr-rate-limit-resource-exhausted-too-many-requests-fails-the-buf-generate.md, design/issues/summary.md]
after: [S-0007]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
usage:
  source: log
  seconds: 2137
  priced_by: harness
  turns:
    - day: 2026-10-29
      work: 29
    - day: 2026-10-30
      ceremony: 1
      work: 17
    - day: 2026-11-03
      ceremony: 1
      work: 5
  models:
    - model: claude-sonnet-5-5
      input: 120
      output: 16623
      cache_read: 3078909
      cache_write: 194474
      cost: 1.2523
  strategic:
    - kind: orchestrator
      seconds: 9
      estimated: true
      models:
        - model: claude-sonnet-5-5
          input: 8
          output: 66
          cache_read: 173654
          cache_write: 76013
          cost: 0.0917
---
# S-0010 Proto publish workflow: BSR push from main and label on proto/v* tags

## Goal

The schema keeps reaching the Buf Schema Registry as `buf.build/bytepunx/signet-proto`: every push to `main` that changes `proto/` runs `buf push` from `proto/`, and a push of `proto/vX.Y.Z` labels the BSR module `vX.Y.Z` and creates a GitHub Release named by the tag, as the old `proto/.github/workflows/publish.yml` and `release.yml` did.

## Acceptance criteria
- [x] `publish-proto-main.yml` pushes to the BSR only when `proto/` changed on `main`
- [x] `release-proto.yml` triggers on `proto/v*` tags, labels the BSR module with the version without the prefix, and creates the GitHub Release with notes from the tag annotation
- [x] `proto/README.md` describes the mono-repo layout, the tag form, and that in-repo consumers generate from the directory while external consumers pin the BSR label

## Tasks
- T-0088 write publish-proto-main.yml: buf push from proto/ when proto/ changes on main
- T-0089 write release-proto.yml: label the BSR module and create the GitHub Release on proto/v* tags
- T-0090 document the mono-repo layout, tag form, and consumption in proto/README.md

## Notes
