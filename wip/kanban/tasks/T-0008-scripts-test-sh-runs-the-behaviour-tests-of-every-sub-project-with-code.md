---
id: T-0008
type: task
nature: improvement
title: scripts/test.sh runs the behaviour tests of every sub-project with code
status: done
parent: S-0004
owner: alex
created: 2026-10-24T22:59:06Z
updated: 2026-11-03T18:20:25Z
transitions:
  - to: ready
    at: 2026-11-03T18:13:09Z
    by: agent-S-0004
  - to: in-progress
    at: 2026-11-03T18:13:09Z
    by: agent-S-0004
  - to: done
    at: 2026-11-03T18:20:25Z
    by: agent-S-0004
stream: S-0004
tags: [scripts, tests]
touches: [scripts/test.sh]
usage:
  source: log
  seconds: 436
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 20
      output: 5221
      cache_read: 1308337
      cache_write: 18796
      cost: 0.5007
---
# T-0008 scripts/test.sh runs the behaviour tests of every sub-project with code

## Work

Replace the placeholder body of `scripts/test.sh` with a POSIX `sh` script under `set -eu` that runs, from the repository root, the behaviour tests of each sub-project with code: `server` through `make -C server test`, `client-go` with `go test -race -count=1 ./...` in `client-go`, `client-python` with `pytest` in `client-python` (its `pyproject.toml` sets `testpaths`), `client-rust` with `cargo test` in `client-rust`, `client-typescript` with `npm test` after `npm run build` in `client-typescript` (its `test` script runs the compiled `dist/**/*.test.js`), and `client-csharp` with `dotnet test SignetClient.sln` in `client-csharp`. Before each project, check its toolchain with `command -v` (`go`, `pytest` or `python3 -m pytest`, `cargo`, `npm`, `dotnet`); when it is absent, print one line naming the project and the missing tool and go on. Run every project even after one fails, collect the failures, print a summary table of project and result, and exit non-zero when any failed. Skip `client-erlang` and `admin`, which have no code, and say so in a comment, not at run time. Quote every variable and keep the script zsh-safe as `tooling.md` says.

## Done when

- `scripts/test.sh` runs the six projects' behaviour tests from any working directory and exits non-zero when any one fails.
- A project whose toolchain is absent is reported as skipped with the tool's name, and the run continues; on this host that is `client-csharp` (`dotnet` is absent).
- `make test` at the root runs it and its output names each project and its result.

## Notes

Verified by running `make test` here: `go`, `cargo`, `pytest`, `npm`, and `buf` are present, `dotnet` is not.
