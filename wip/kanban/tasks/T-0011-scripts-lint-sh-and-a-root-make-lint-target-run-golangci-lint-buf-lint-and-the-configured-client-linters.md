---
id: T-0011
type: task
nature: improvement
title: scripts/lint.sh and a root make lint target run golangci-lint, buf lint, and the configured client linters
status: done
parent: S-0004
owner: alex
created: 2026-10-24T22:59:34Z
updated: 2026-11-03T18:20:38Z
transitions:
  - to: ready
    at: 2026-11-03T18:13:09Z
    by: agent-S-0004
  - to: in-progress
    at: 2026-11-03T18:13:09Z
    by: agent-S-0004
  - to: done
    at: 2026-11-03T18:20:38Z
    by: agent-S-0004
stream: S-0004
tags: [scripts, lint, make]
touches: [scripts/lint.sh, Makefile]
usage:
  source: log
  seconds: 449
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 30
      output: 7800
      cache_read: 1954583
      cache_write: 28080
      cost: 0.748
---
# T-0011 scripts/lint.sh and a root make lint target run golangci-lint, buf lint, and the configured client linters

## Work

Add `scripts/lint.sh`, POSIX `sh` under `set -eu`, that runs from the repository root: `make -C server lint` (`golangci-lint run ./...` with `server/.golangci.yml`), `golangci-lint run ./...` in `client-go` using the same `server/.golangci.yml` through `--config`, and `buf lint` in `proto`. Then each client's linter where one is configured: `cargo fmt --check` and `cargo clippy --all-targets -- -D warnings` in `client-rust` (both ship with cargo), `tsc --noEmit -p tsconfig.json` in `client-typescript` (no eslint is configured), and nothing for `client-python` and `client-csharp`, which configure no linter today; print one line for each saying so. Check each tool with `command -v` and report a skip when it is absent, as `scripts/test.sh` does. Run every linter even after one fails and exit non-zero when any failed. Add a `lint` target to the root `Makefile` that calls `scripts/lint.sh`, with a `##` comment so `make help` lists it. Leave `lint-md` as it is.

## Done when

- `make lint` at the root runs golangci-lint for `server` and `client-go`, `buf lint` for `proto`, and the Rust and TypeScript checks, and exits non-zero when any fails.
- A client with no configured linter is named with that statement; an absent tool is reported as a skip.
- `make help` lists `lint`.

## Notes

Whether to add ruff or eslint configurations to the clients is a change to those projects, not this story's.
