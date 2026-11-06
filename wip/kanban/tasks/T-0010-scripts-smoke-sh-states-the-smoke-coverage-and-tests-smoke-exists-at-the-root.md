---
id: T-0010
type: task
nature: improvement
title: scripts/smoke.sh states the smoke coverage and tests/smoke/ exists at the root
status: done
parent: S-0004
owner: alex
created: 2026-10-24T22:59:25Z
updated: 2026-11-03T18:20:36Z
transitions:
  - to: ready
    at: 2026-11-03T18:13:09Z
    by: agent-S-0004
  - to: in-progress
    at: 2026-11-03T18:13:09Z
    by: agent-S-0004
  - to: done
    at: 2026-11-03T18:20:36Z
    by: agent-S-0004
stream: S-0004
tags: [scripts, tests]
touches: [scripts/smoke.sh, tests/smoke/README.md]
usage:
  source: log
  seconds: 447
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 10
      output: 2547
      cache_read: 638212
      cache_write: 9169
      cost: 0.2442
---
# T-0010 scripts/smoke.sh states the smoke coverage and tests/smoke/ exists at the root

## Work

Create `tests/smoke/README.md` with front matter (`title`, `updated`, `status`) saying what the folder is for per `code-quality.md`: end-to-end paths through the full system from the consumer's perspective, run with `make smoke`. State what smoke coverage exists today and where it will come from: the five `client-*/examples/echo/` images (`Dockerfile` in each) were the signet-clients `smoke-images.yml` harness, which the import dropped, and S-0005 rebuilds CI as root workflows; `bytepunx/kluster` (k3d) is the cluster the conventions name for local runs. Replace the placeholder body of `scripts/smoke.sh` with a POSIX `sh` script under `set -eu` that runs any executable test script under `tests/smoke/` when one exists, and otherwise prints one line pointing at `tests/smoke/README.md` for the coverage statement and exits 0, so the tier reports its state plainly rather than failing for want of tests.

## Done when

- `tests/smoke/README.md` exists with front matter and names the existing echo images and the story that brings their workflow.
- `scripts/smoke.sh` runs the scripts under `tests/smoke/` when present and otherwise says plainly that the tier has no tests and where the coverage will come from.
- `make smoke` at the root runs it and exits 0 on this repository.

## Notes

The folder gets a README rather than a placeholder test so that `flai check` and `documentation.md`'s folder rule are both met.
