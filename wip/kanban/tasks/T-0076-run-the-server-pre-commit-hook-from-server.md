---
id: T-0076
type: task
nature: remediation
title: Run the server pre-commit hook from server/
status: in-progress
parent: S-0017
owner: alex
created: 2026-10-27T22:52:47Z
updated: 2026-11-06T08:12:48Z
transitions:
  - to: ready
    at: 2026-11-06T08:12:48Z
    by: agent-S-0017
  - to: in-progress
    at: 2026-11-06T08:12:48Z
    by: agent-S-0017
stream: S-0017
tags: [server]
touches: [server/githooks/pre-commit, server/Makefile]
---
# T-0076 Run the server pre-commit hook from server/

## Work

The hook does `cd "$(git rev-parse --show-toplevel)"`, which is now the repository root, so `gofmt -l .` and `golangci-lint run ./...` run over the whole mono-repo, and `server/Makefile`'s `install-hooks` sets `core.hooksPath githooks`, a path that is not at the root.

- In `server/githooks/pre-commit`, change into `server/` (`cd "$(git rev-parse --show-toplevel)/server"`), so that it checks the Go module of `server/` and nothing else. Keep the hook POSIX `sh` under `set -e`. Skip the checks when the commit stages no file under `server/` (use `git diff --cached --name-only`), so a commit that changes only docs or another component does not run the Go lint.
- In `server/Makefile`, make `install-hooks` set `core.hooksPath` to `server/githooks` from the repository root (`git config core.hooksPath server/githooks`, run from `git rev-parse --show-toplevel` since the target runs from `server/`). Update the echo line.
- Check the hook's comment ("same as CI's Lint job") against the lint job in `.github/workflows/` and keep it true.

## Done when

- Running the hook from the repository root with a staged change under `server/` formats and lints only `server/`.
- Running it with no staged file under `server/` exits 0 at once.
- `make -C server install-hooks` sets `core.hooksPath` to `server/githooks`, as `git config core.hooksPath` read from the root confirms.
- `flai check --strict` passes.

## Notes

Layer 1: it shares no path with another task. Undo the `core.hooksPath` setting after the check if it was unset before.
