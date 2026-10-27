---
id: T-0001
type: task
nature: feature
title: Scaffold client-elixir Mix project, Docker toolchain, and stub generation
status: done
parent: S-0001
owner: alex
created: 2026-10-27T19:07:28Z
updated: 2026-10-27T19:10:43Z
transitions:
  - to: ready
    at: 2026-10-27T19:07:47Z
    by: agent
  - to: in-progress
    at: 2026-10-27T19:08:46Z
    by: agent
  - to: done
    at: 2026-10-27T19:10:43Z
    by: agent
stream: S-0001
tags: []
touches: [client-elixir/mix.exs, client-elixir/Dockerfile, client-elixir/buf.gen.yaml, client-elixir/buf.gen.spiffe.yaml, client-elixir/proto/spiffe/workload/workload.proto, client-elixir/.gitignore, client-elixir/.dockerignore, client-elixir/.formatter.exs, client-elixir/mix.lock, client-elixir/lib/signet.ex, client-elixir/test/test_helper.exs, scripts/test-client-elixir.sh, scripts/test.sh, scripts/README.md, system-flow.yaml]
---
# T-0001 Scaffold client-elixir Mix project, Docker toolchain, and stub generation

## Work

Create `client-elixir/` as a Mix library `signet_client` with `grpc`, `gun`, and `protobuf` dependencies, a `buf.gen.yaml` that generates `signet.v1` and `admin.v1` stubs from `buf.build/bytepunx/signet-proto:v0.4.0` with the local `protoc-gen-elixir` plugin, a second template for the vendored SPIFFE Workload API proto under `proto/`, a Dockerfile pinning Elixir, Erlang, buf, and the plugin, and `scripts/test-client-elixir.sh` that builds the image and runs generation, compile, and tests. Wire it into `scripts/test.sh` and register the folder in `system-flow.yaml`.

## Done when

`scripts/test-client-elixir.sh` builds from a clean checkout in Docker, generates stubs that compile, and runs an empty test suite green. Generated code is gitignored.

## Notes
