---
id: T-0009
type: task
nature: improvement
title: scripts/integration.sh runs the server's -tags integration tests and reports the clients' tier
status: done
parent: S-0004
owner: alex
created: 2026-10-24T22:59:15Z
updated: 2026-11-03T18:20:33Z
transitions:
  - to: ready
    at: 2026-11-03T18:13:09Z
    by: agent-S-0004
  - to: in-progress
    at: 2026-11-03T18:13:09Z
    by: agent-S-0004
  - to: done
    at: 2026-11-03T18:20:33Z
    by: agent-S-0004
stream: S-0004
tags: [scripts, tests]
touches: [scripts/integration.sh]
usage:
  source: log
  seconds: 444
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 15
      output: 3900
      cache_read: 977233
      cache_write: 14039
      cost: 0.374
---
# T-0009 scripts/integration.sh runs the server's -tags integration tests and reports the clients' tier

## Work

Replace the placeholder body of `scripts/integration.sh` with a POSIX `sh` script under `set -eu` that runs the server's integration tier through `make -C server test-int`, which is `go test -race -count=1 -tags integration ./...`. Those tests, under `server/internal/store/*_integration_test.go`, start a `postgres:16-alpine` container with testcontainers, so check for `go` and for a reachable Docker daemon (`docker info`) first and fail with one line saying which is missing, rather than letting testcontainers panic. For each client, look for the folder `code-quality.md` names, `<project>/tests/integration/`; none exists today, so print one line per client saying its integration tier has no tests yet, and run the folder's tests when one appears later (Go, pytest, cargo, npm, dotnet, as `scripts/test.sh` does). Exit non-zero when the server tier fails.

## Done when

- `scripts/integration.sh` runs the server's `-tags integration` tests and exits with their status.
- With Docker or `go` absent it stops with one line naming what is missing.
- Each client without `tests/integration/` is named with the statement that its tier has no tests.
- `make integration` at the root runs it.

## Notes

The server's integration tests live beside the code behind a build tag, not under `server/tests/integration/`; moving them is not this story's work.
