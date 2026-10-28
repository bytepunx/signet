---
id: T-0042
type: task
nature: feature
title: Shared buf generate script that runs in Docker
status: done
parent: S-0007
owner: alex
created: 2026-10-27T22:26:07Z
updated: 2026-10-27T23:04:43Z
transitions:
  - to: ready
    at: 2026-10-27T22:57:53Z
    by: agent-S-0007
  - to: in-progress
    at: 2026-10-27T22:57:53Z
    by: agent-S-0007
  - to: done
    at: 2026-10-27T23:04:43Z
    by: agent-S-0007
stream: S-0007
tags: [ci]
touches: [scripts/buf-generate.sh, ".gitignore"]
usage:
  source: log
  seconds: 410
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 9
      output: 1492
      cache_read: 416360
      cache_write: 17681
      cost: 0.1113
---
# T-0042 Shared buf generate script that runs in Docker

## Work

Add `scripts/buf-generate.sh <component> [buf args]`, POSIX `sh` under `set -eu`, that runs `buf generate` for one component inside the pinned `bufbuild/buf` image (1.71.0, the version `client-elixir/Dockerfile` pins, recorded in `design/tech/buf.md`), so no script installs buf on a host or a runner.

- Mount the repository root at `/repo` and run in `/repo/<component>`, so a `buf.gen.yaml` input of either a BSR module (today) or `../proto/proto` (story S-0020) resolves.
- Run as the caller's user (`--user "$(id -u):$(id -g)"`) and set `HOME` and `BUF_CACHE_DIR` to a writable path, so generated files are not root-owned on a Linux runner and the cache survives between runs.
- Pass `BUF_TOKEN` through when set, so remote plugins are not rate limited anonymously.
- Print what it does and exit non-zero on failure, as `scripts/README.md` asks.

The per-component test scripts (layer 2) call this script instead of repeating the `docker run`.

## Done when

- `scripts/buf-generate.sh client-python` leaves the stubs `client-python/buf.gen.yaml` names under `client-python/src/`, gitignored, and exits 0.
- `scripts/buf-generate.sh client-go` leaves `client-go/gen` with no diff from what is committed (`git diff --quiet client-go/gen`).
- A bad component name exits non-zero with a message.
- `sh -n scripts/buf-generate.sh` is clean and the file is executable.

## Notes

Layer 1 of the plan: waits for nothing.
