---
id: T-0071
type: task
nature: feature
title: Script that builds the echo image of a client
status: done
parent: S-0019
owner: alex
created: 2026-10-27T22:52:40Z
updated: 2026-11-06T08:47:53Z
transitions:
  - to: ready
    at: 2026-11-06T08:47:49Z
    by: agent-S-0019
  - to: in-progress
    at: 2026-11-06T08:47:49Z
    by: agent-S-0019
  - to: done
    at: 2026-11-06T08:47:53Z
    by: agent-S-0019
stream: S-0019
tags: [ci]
touches: [scripts/build-echo-images.sh]
usage:
  source: log
  seconds: 4
  estimated: true
  priced_by: harness
  models: []
---
# T-0071 Script that builds the echo image of a client

## Work

Add `scripts/build-echo-images.sh`, POSIX `sh` under `set -eu`, safe to run from any directory. `build-echo-images.sh <language>... | all [--tag <tag>] [--push]` builds `ghcr.io/bytepunx/signet-echo-<language>:<tag>` (default tag `local`) for each of `go`, `python`, `typescript`, `rust`, and `csharp` from `client-<language>/examples/echo/Dockerfile`. It takes the build command from each Dockerfile's own header comment, read after S-0020 changed them: the context is `client-<language>/` or the repository root, and the script passes `-f` and the context the Dockerfile asks for. `--push` pushes the tag; without it nothing leaves the host. An unknown language, or `elixir` while `client-elixir/examples/echo/Dockerfile` does not exist, exits non-zero naming it. The script prints each `docker build` it runs.

It is one script so that the workflow (`.github/workflows/smoke-images.yml`) and `scripts/smoke.sh` build the images the same way, as `tooling.md` asks of scripts and CI.

This task waits for no other task: it is the first layer. The story itself waits on S-0020, which rewrites the python, rust, and csharp echo Dockerfiles' build context.

## Done when

- `scripts/build-echo-images.sh go` builds `ghcr.io/bytepunx/signet-echo-go:local` with Docker, and the same for each of the other four languages, checked by running them.
- `scripts/build-echo-images.sh all --tag test` builds the five and pushes none.
- `scripts/build-echo-images.sh elixir` and `scripts/build-echo-images.sh nope` exit non-zero with a message that names the language.
- The script is executable and passes `shellcheck` where it is installed.

## Notes

Dockerfile build contexts differ per language and S-0020 moves three of them to the repository root, so the script reads them rather than hard-coding them. The images are test-only: each Dockerfile says it prints secrets to stdout.
