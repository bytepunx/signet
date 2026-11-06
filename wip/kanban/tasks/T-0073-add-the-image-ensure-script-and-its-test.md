---
id: T-0073
type: task
nature: improvement
title: Add the image-ensure script and its test
status: done
parent: S-0021
owner: alex
created: 2026-10-27T22:52:43Z
updated: 2026-11-06T08:02:49Z
transitions:
  - to: ready
    at: 2026-11-06T08:02:05Z
    by: agent-S-0021
  - to: in-progress
    at: 2026-11-06T08:02:05Z
    by: agent-S-0021
  - to: done
    at: 2026-11-06T08:02:49Z
    by: agent-S-0021
stream: S-0021
tags: [scripts]
touches: [scripts/ensure-client-elixir-image.sh, scripts/test-ensure-client-elixir-image.sh]
usage:
  source: log
  seconds: 44
  estimated: true
  priced_by: estimate
  models:
    - model: claude-sonnet-5-5
      input: 12
      output: 94
      cache_read: 445440
      cache_write: 9553
      cost: 0.1524
---
# T-0073 Add the image-ensure script and its test

## Work

Write `scripts/ensure-client-elixir-image.sh` (POSIX sh, `set -eu`, safe from any directory). It computes the SHA-256 of `client-elixir/Dockerfile` (`sha256sum`, else `shasum -a 256`), reads the label `signet.dockerfile-sha256` from `signet-client-elixir:toolchain` with `docker image inspect --format`, and runs `docker build -q --label signet.dockerfile-sha256=<hash> -t signet-client-elixir:toolchain client-elixir` only when the image is missing, the label differs, or `SIGNET_REBUILD=1`. It prints which case it took (skipped, built) so a log shows it. `docker image inspect` only talks to the local daemon, so the skip path never resolves the base image on the registry.

Write `scripts/test-ensure-client-elixir-image.sh`, which reproduces I-0002: it puts a fake `docker` script first on `PATH` in a temp directory, records each call, and asserts that the build is not run when the label matches, and is run when the image is missing, when the label differs, and when `SIGNET_REBUILD=1`.

## Done when

- `scripts/test-ensure-client-elixir-image.sh` passes and fails when the skip condition in the ensure script is broken.
- Running the ensure script twice against the real daemon builds once and skips the second time.
- Both scripts pass `flai check --strict`.

## Notes
