---
id: S-0020
type: story
nature: improvement
title: Generate in-repo stubs from proto/ by directory instead of the BSR pin
status: ready
parent: E-0002
owner: arobson
created: 2026-10-27T19:48:36Z
updated: 2026-11-06T08:37:57Z
transitions:
  - to: ready
    at: 2026-10-27T23:00:58Z
    by: alex
tags: [server, proto]
topics: [proto]
touches: [server/buf.gen.yaml, client-go/buf.gen.yaml, client-go/README.md, client-python/buf.gen.yaml, client-python/README.md, client-python/examples/echo/Dockerfile, client-python/examples/echo/Dockerfile.dockerignore, client-typescript/buf.gen.yaml, client-typescript/README.md, client-rust/buf.gen.yaml, client-rust/README.md, client-rust/examples/echo/Dockerfile, client-csharp/buf.gen.yaml, client-csharp/README.md, client-csharp/examples/echo/Dockerfile, client-csharp/examples/echo/Dockerfile.dockerignore, client-elixir/buf.gen.yaml, client-elixir/README.md, scripts/test-client-elixir.sh, design/tech/buf.md]
after: [S-0005, S-0007, S-0011]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
usage:
  source: sum
  seconds: 0
  models: []
  strategic:
    - kind: planner
      seconds: 179
      estimated: true
      models:
        - model: claude-haiku-5-5
          input: 30
          output: 94
          cache_read: 810527
          cache_write: 87952
          cost: 0
        - model: claude-sonnet-5-5
          input: 200
          output: 31268
          cache_read: 7958441
          cache_write: 404381
          cost: 4.004
cost_of_delay:
  inputs:
    time_lost_per_cycle: 30m
    by: alex
    at: 2026-11-06T08:31:51Z
forecast:
  duration: 3h
  delivery: 2026-11-06T22:19:00Z
  basis: "Its own forecast of 3h; 1st in the pull order with an in-progress limit of 4, with nothing ahead of it."
  by: flai
  at: 2026-11-06T08:37:57Z
---
# S-0020 Generate in-repo stubs from proto/ by directory instead of the BSR pin

## Goal

Every in-repo consumer of the schema (the server and the six clients) generates its stubs from `../proto/proto` as a directory input in its `buf.gen.yaml`, so a schema change and the code that uses it land in one pull request and the `proto-bump.yml` automation of the old repository is not needed. The BSR pin stays only in documentation for external consumers.

## Acceptance criteria
- [ ] Each `buf.gen.yaml` names `directory: ../proto/proto` as its input and `buf generate` produces stubs identical to those from the `v0.4.0` BSR pin at the same schema
- [ ] `make test` passes for every component
- [ ] Each component's README no longer tells readers to bump a BSR pin

## Tasks
- T-0028 Server: generate gen/ from ../proto/proto
- T-0029 Go client: generate gen/ from ../proto/proto
- T-0031 Python client: generate src/ stubs from ../proto/proto and fix the echo image build
- T-0032 TypeScript client: generate src/gen from ../proto/proto
- T-0034 Rust client: generate src/gen from ../proto/proto and fix the echo image build
- T-0035 C# client: generate gen/ from ../proto/proto and fix the echo image build
- T-0036 Elixir client: generate lib/gen from ../proto/proto and mount the repository in the test script
- T-0038 Tech record: in-repo consumers generate from the directory, and the final cross-component check

## Notes

Touches every component, so publishing it releases a patch of each; that is correct, since each ships regenerated stubs. Tagged `server` and `proto`: a story touching several components must name the one it delivers to; the others get a patch. Depends on the ADR deciding directory input.

### Planning

Touches: the seven `buf.gen.yaml` files were declared. Layout added each component's README (all but the server's say "pinned" and `buf generate` pulls from `buf.build/bytepunx/signet-proto`) and design added `design/tech/buf.md`. Layout and a read of the echo Dockerfiles added the python, rust, and csharp `examples/echo/Dockerfile` files (and the python and csharp `Dockerfile.dockerignore`): each copies only `buf.gen.yaml` into a stage and runs `buf generate`, which a relative directory input breaks, so they build from the repository root instead. `scripts/test-client-elixir.sh` mounts only `client-elixir/`, so it must mount the repository root. `flai touches suggest` listed only co-change from generated files (`gen/` stubs, `package.json`), which these changes should leave unchanged, so none were added. No folder touches.

Left out on purpose: `server/CLAUDE.md` (S-0017 rewrites it), `server/.github/workflows/ci.yml` (S-0006 removes it and S-0007's root CI runs the drift check), `proto/README.md` (S-0010 owns it; it documents the BSR for external consumers, which stays), and the TypeScript echo Dockerfile, which builds from the committed `src/gen`.

After: S-0005 (the ADR deciding directory input), S-0007 (the root CI and `scripts/test-*.sh` that criterion 2's `make test` runs for every component), and S-0011 (it changes the Go client's module path and `go_package_prefix` in `client-go/buf.gen.yaml`, and the regenerated Go stubs must match it). S-0011 already follows S-0007.

Forecast: 3h, up from the 1h default for size 10. Seven independent generate-and-diff runs against remote plugins, three Dockerfile rewrites, and a script change. Delivery adds the 2h to the computed 2026-10-28T01:25:00Z; it also waits on S-0005, S-0007, and S-0011.

Cost of delay: not worked out. The story has no inputs and E-0002 has none; asked on the plan thread.
