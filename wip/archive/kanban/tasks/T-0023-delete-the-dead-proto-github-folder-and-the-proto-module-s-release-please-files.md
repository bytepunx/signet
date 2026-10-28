---
id: T-0023
type: task
nature: improvement
title: Delete the dead proto/.github folder and the proto module's release-please files
status: done
parent: S-0006
owner: alex
created: 2026-10-27T22:25:18Z
updated: 2026-10-27T22:57:00Z
transitions:
  - to: ready
    at: 2026-10-27T22:56:55Z
    by: agent-S-0006
  - to: in-progress
    at: 2026-10-27T22:56:55Z
    by: agent-S-0006
  - to: done
    at: 2026-10-27T22:57:00Z
    by: agent-S-0006
stream: S-0006
tags: [proto]
touches: [proto/.github/workflows/ci.yml, proto/.github/workflows/publish.yml, proto/.github/workflows/release-please.yml, proto/.github/workflows/release.yml, proto/release-please-config.json, proto/.release-please-manifest.json]
usage:
  source: log
  seconds: 5
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 1
      output: 129
      cache_read: 24953
      cache_write: 1815
      cost: 0.011
---
# T-0023 Delete the dead proto/.github folder and the proto module's release-please files

## Work

Delete, with `git rm`, every file under `proto/.github/` (CI, Publish, Release Please, and Release workflows; GitHub does not run them from a sub-folder) and `proto/release-please-config.json` and `proto/.release-please-manifest.json`. Leave `proto/CHANGELOG.md`, which records the releases release-please made.

Before deleting, check that nothing else reads the deleted files, in `proto/` and at the root. Leave the sentence in `proto/README.md` that says releases are tagged via release-please to S-0010, which rewrites that README for the BSR workflow.

The deleted workflows stay in git history as the reference for S-0010.

## Done when

- `proto/.github/` does not exist.
- `proto/release-please-config.json` and `proto/.release-please-manifest.json` do not exist.
- No remaining file under `proto/` loads or runs a deleted file.
- `flai check --strict` is clean.

## Notes

Layer 1. It shares no path with the other tasks of S-0006, so it runs together with them.
