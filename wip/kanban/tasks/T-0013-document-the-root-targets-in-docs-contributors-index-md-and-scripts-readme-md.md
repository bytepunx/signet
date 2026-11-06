---
id: T-0013
type: task
nature: improvement
title: Document the root targets in docs/contributors/index.md and scripts/README.md
status: done
parent: S-0004
owner: alex
created: 2026-10-24T23:00:06Z
updated: 2026-11-03T18:25:34Z
transitions:
  - to: ready
    at: 2026-11-03T18:25:06Z
    by: agent-S-0004
  - to: in-progress
    at: 2026-11-03T18:25:06Z
    by: agent-S-0004
  - to: done
    at: 2026-11-03T18:25:34Z
    by: agent-S-0004
stream: S-0004
tags: [docs, scripts]
touches: [docs/contributors/index.md, scripts/README.md]
after: [T-0008, T-0009, T-0010, T-0011, T-0012, T-0019]
usage:
  source: log
  seconds: 28
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 5
      output: 1412
      cache_read: 353934
      cache_write: 5085
      cost: 0.1354
---
# T-0013 Document the root targets in docs/contributors/index.md and scripts/README.md

## Work

Rewrite `docs/contributors/index.md` (bump `updated`, set `status: active`) with a table of the root `make` targets and what each runs: `test`, `integration`, `smoke`, `lint`, `lint-md`, `build`, `install-hooks`, `check`, plus the toolchain each needs and the fact that a project is skipped, saying so, when its toolchain is absent. Name `server/Makefile` as the per-project entry point the root scripts call, and `tests/smoke/README.md` for the smoke coverage statement. Add rows for `test.sh` (its six projects), `integration.sh`, `smoke.sh`, `lint.sh`, `build.sh`, and `install-hooks.sh` to the table in `scripts/README.md`, replacing the three placeholder rows. Link documents by relative path. Waits for the five script tasks because it describes what they built.

## Done when

- `docs/contributors/index.md` lists every root target with what it runs and its toolchain.
- `scripts/README.md`'s table has one row per script in `scripts/`.
- `make lint-md` and `flai check --strict` are clean.

## Notes

Operator and user documentation move in S-0009; this task touches the contributors' guide only.
