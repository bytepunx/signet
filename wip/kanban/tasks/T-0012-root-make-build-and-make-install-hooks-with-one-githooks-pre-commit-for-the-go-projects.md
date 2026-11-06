---
id: T-0012
type: task
nature: improvement
title: Root make build and make install-hooks, with one githooks/pre-commit for the Go projects
status: done
parent: S-0004
owner: alex
created: 2026-10-24T22:59:53Z
updated: 2026-11-03T18:24:25Z
transitions:
  - to: ready
    at: 2026-11-03T18:21:14Z
    by: agent-S-0004
  - to: in-progress
    at: 2026-11-03T18:21:14Z
    by: agent-S-0004
  - to: done
    at: 2026-11-03T18:24:25Z
    by: agent-S-0004
stream: S-0004
tags: [scripts, make, githooks]
touches: [scripts/build.sh, scripts/install-hooks.sh, scripts/golangci-lint.sh, scripts/lint.sh, githooks/pre-commit, Makefile, server/Makefile, server/githooks/pre-commit]
after: [T-0011]
usage:
  source: log
  seconds: 191
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 32
      output: 8282
      cache_read: 2075536
      cache_write: 29818
      cost: 0.7943
---
# T-0012 Root make build and make install-hooks, with one githooks/pre-commit for the Go projects

## Work

Add `scripts/build.sh`, POSIX `sh` under `set -eu`, that runs `make -C server build`, so `signetd` and `signet` land in `server/bin` (already ignored by the root `.gitignore`'s `bin/`). Add `githooks/pre-commit` at the root: the body of `server/githooks/pre-commit` (gofmt check, golangci-lint presence check, `golangci-lint run ./...`) run in each Go module, `server` and `client-go`, from the repository top level, with `client-go` using `server/.golangci.yml` through `--config` as `scripts/lint.sh` does, plus whatever the signet-clients `.githooks/pre-commit` ran for the other clients. The import dropped that file: read it from the signet-clients repository (`github.com/bytepunx/signet-clients`, `.githooks/`) with a read-only fetch or the forge's web view; if it is unreachable, open a thread on this task saying so and ship the Go checks alone. Add `scripts/install-hooks.sh` that runs `git config core.hooksPath githooks` from the root and prints what the hook runs, and point `server/Makefile`'s `install-hooks` at it (`$(MAKE) -C .. install-hooks`) rather than at a `githooks` path git resolves against the root, where none existed; remove `server/githooks/pre-commit` once the root hook covers it. Add `build` and `install-hooks` targets to the root `Makefile`, each with a `##` comment. Waits for T-0011 because both edit the root `Makefile`.

## Done when

- `make build` at the root produces `server/bin/signetd` and `server/bin/signet`.
- `make install-hooks` at the root sets `core.hooksPath` to `githooks`, and `githooks/pre-commit` runs gofmt and golangci-lint for `server` and `client-go` and the checks the signet-clients hook ran, or a thread on this task records that the old hook could not be read.
- `make -C server install-hooks` installs the same root hook; `server/githooks/` is gone.
- `make help` lists `build` and `install-hooks`.

## Notes

`server/Makefile` stays the per-project entry point; only its `install-hooks` target changes.
