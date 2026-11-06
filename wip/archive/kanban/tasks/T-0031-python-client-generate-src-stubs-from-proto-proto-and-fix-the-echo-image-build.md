---
id: T-0031
type: task
nature: improvement
title: "Python client: generate src/ stubs from ../proto/proto and fix the echo image build"
status: done
parent: S-0020
owner: alex
created: 2026-10-27T22:25:35Z
updated: 2026-11-06T08:41:11Z
transitions:
  - to: ready
    at: 2026-11-06T08:38:26Z
    by: agent-S-0020
  - to: in-progress
    at: 2026-11-06T08:38:26Z
    by: agent-S-0020
  - to: done
    at: 2026-11-06T08:41:11Z
    by: agent-S-0020
stream: S-0020
tags: [client-python, proto]
touches: [client-python/buf.gen.yaml, client-python/README.md, client-python/examples/echo/Dockerfile, client-python/examples/echo/Dockerfile.dockerignore]
usage:
  source: log
  seconds: 165
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 4
      output: 847
      cache_read: 153090
      cache_write: 5931
      cost: 0.0475
---
# T-0031 Python client: generate src/ stubs from ../proto/proto and fix the echo image build

## Work

Replace the pinned BSR module input in `client-python/buf.gen.yaml` with `directory: ../proto/proto` and rewrite its comment. The echo image's `proto` stage copies only `buf.gen.yaml` and runs `buf generate`, which a relative directory input breaks: change the Dockerfile to build from the repository root (`docker build -f client-python/examples/echo/Dockerfile .`), copy `proto/proto` and `client-python/buf.gen.yaml` to the same relative layout the repository has, and run `buf generate` there. Update `Dockerfile.dockerignore` and the build comment to match, and drop the claim that the build needs network access to buf.build for the schema (the remote plugins still need it). In `client-python/README.md` rewrite the opening paragraph and the two `buf generate` sections (about lines 35 to 43 and 214) so they no longer tell readers to bump a BSR pin.

## Done when

- `client-python/buf.gen.yaml` names `directory: ../proto/proto` as its only input.
- `buf generate` in `client-python/` writes the same files under `src/signet` and `src/admin` as the `v0.4.0` pin did.
- The echo image builds from the repository root with the documented command.
- The client's tests pass.
- `client-python/README.md` no longer tells readers to bump a BSR pin.

## Notes
