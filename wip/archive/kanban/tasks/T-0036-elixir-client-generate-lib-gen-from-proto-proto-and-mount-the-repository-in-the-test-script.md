---
id: T-0036
type: task
nature: improvement
title: "Elixir client: generate lib/gen from ../proto/proto and mount the repository in the test script"
status: done
parent: S-0020
owner: alex
created: 2026-10-27T22:25:47Z
updated: 2026-11-06T08:41:13Z
transitions:
  - to: ready
    at: 2026-11-06T08:38:27Z
    by: agent-S-0020
  - to: in-progress
    at: 2026-11-06T08:38:27Z
    by: agent-S-0020
  - to: done
    at: 2026-11-06T08:41:13Z
    by: agent-S-0020
stream: S-0020
tags: [client-elixir, proto]
touches: [client-elixir/buf.gen.yaml, client-elixir/README.md, scripts/test-client-elixir.sh]
usage:
  source: log
  seconds: 166
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 5
      output: 967
      cache_read: 174860
      cache_write: 6775
      cost: 0.0543
---
# T-0036 Elixir client: generate lib/gen from ../proto/proto and mount the repository in the test script

## Work

Replace the pinned BSR module input in `client-elixir/buf.gen.yaml` with `directory: ../proto/proto` and rewrite its comment. `buf.gen.spiffe.yaml` already reads the vendored `proto` directory and stays as it is. `scripts/test-client-elixir.sh` mounts only `client-elixir/` at `/app`, so `../proto/proto` does not exist in the container: mount the repository root and run each command with the working directory `client-elixir`, keeping `deps/`, `_build/`, and `lib/gen/` on the host under `client-elixir/`. In `client-elixir/README.md` rewrite the opening paragraph and the generate section (about lines 4 and 39 to 50), which says the first template pulls from `buf.build/bytepunx/signet-proto`, so they say it reads `../proto/proto` and no longer tell readers to bump a BSR pin.

## Done when

- `client-elixir/buf.gen.yaml` names `directory: ../proto/proto` as its only input.
- `scripts/test-client-elixir.sh` passes, so `make test` passes for the Elixir client.
- The `lib/gen/` modules it generates match those the `v0.4.0` pin produced.
- `client-elixir/README.md` no longer tells readers to bump a BSR pin.

## Notes
