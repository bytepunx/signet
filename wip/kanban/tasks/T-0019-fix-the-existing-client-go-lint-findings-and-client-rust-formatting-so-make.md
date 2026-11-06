---
id: T-0019
type: task
nature: remediation
title: Fix the existing client-go lint findings and client-rust formatting so make lint passes
status: done
parent: S-0004
owner: alex
created: 2026-11-03T18:21:08Z
updated: 2026-11-03T18:24:21Z
transitions:
  - to: ready
    at: 2026-11-03T18:21:14Z
    by: agent-S-0004
  - to: in-progress
    at: 2026-11-03T18:21:14Z
    by: agent-S-0004
  - to: done
    at: 2026-11-03T18:24:21Z
    by: agent-S-0004
stream: S-0004
tags: []
touches: [client-go/restart_test.go, client-go/client_test.go, client-rust/src/client.rs, client-rust/src/lib.rs, client-rust/src/restart.rs, client-rust/examples/echo.rs, client-rust/examples/getsecret.rs, client-rust/examples/restart_on_change.rs]
usage:
  source: log
  seconds: 187
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 27
      output: 6859
      cache_read: 1718954
      cache_write: 24695
      cost: 0.6578
---
# T-0019 Fix the existing client-go lint findings and client-rust formatting so make lint passes

## Work

`make lint` fails on code that predates this story, and the root pre-commit hook (T-0012) runs the same Go checks, so it would block every commit until they pass. Fix the five golangci-lint findings in `client-go` under `server/.golangci.yml`: the unchecked `lock.Release` errors in `restart_test.go` (lines 120, 157, 177, 226) and the `defer conn.Close()` before a return in `client_test.go:123`. Run `cargo fmt` in `client-rust` so `cargo fmt --check` passes. Change behaviour nowhere: test files and formatting only.

## Done when

- `scripts/lint.sh` reports `client-go golangci-lint` and `client-rust cargo fmt` as passed.
- `client-go` and `client-rust` tests still pass.

## Notes

Found by T-0011. Runs beside T-0012: they share no path.
