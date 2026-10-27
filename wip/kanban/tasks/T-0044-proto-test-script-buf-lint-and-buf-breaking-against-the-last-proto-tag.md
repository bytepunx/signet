---
id: T-0044
type: task
nature: feature
title: "Proto test script: buf lint and buf breaking against the last proto tag"
status: backlog
parent: S-0007
owner: alex
created: 2026-10-27T22:26:10Z
updated: 2026-10-27T22:26:10Z
transitions: []
stream: S-0007
tags: [proto, ci]
touches: [scripts/test-proto.sh]
---
# T-0044 Proto test script: buf lint and buf breaking against the last proto tag

## Work

Add `scripts/test-proto.sh`, POSIX `sh` under `set -eu`, that runs from `proto/` inside the pinned `bufbuild/buf` image (1.71.0, as `design/tech/buf.md` records):

1. `buf lint`.
2. `buf breaking` against the last `proto/v*` tag, as the old `proto/.github/workflows/ci.yml` did against `v*`: find the tag with `git tag --list 'proto/v*' --sort=-v:refname`, and compare with `--against "<repo>/.git#tag=<tag>,subdir=proto"`, mounting the repository root so `.git` is readable (set `safe.directory` for the container user). When no `proto/v*` tag exists yet, print that and skip the breaking check without failing, since story S-0006 creates the baseline tags and a clone may not have fetched tags.

In CI the checkout needs `fetch-depth: 0` for the tags; the ci.yml task sets it.

## Done when

- `scripts/test-proto.sh` exits 0 on the current `proto/` and prints the lint result and either the breaking comparison or the skip.
- With a `proto/v*` tag present in a scratch clone, a breaking change to a copy of a `.proto` file makes it exit non-zero.
- `sh -n scripts/test-proto.sh` is clean and the file is executable.

## Notes

Layer 1 of the plan: waits for nothing. It does not call `scripts/buf-generate.sh`, since it generates nothing.
