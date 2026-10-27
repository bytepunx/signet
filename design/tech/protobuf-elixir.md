---
title: protobuf and protoc-gen-elixir
updated: 2026-10-27
status: active
topics: [client-elixir, proto]
---

# protobuf and protoc-gen-elixir

| | |
|---|---|
| Version | `protobuf` 0.17.1, the same release as the `protoc-gen-elixir` escript |
| Used in | `client-elixir/`: runtime for `lib/gen/`, and the generator `buf generate` runs |
| Pinned in | `client-elixir/mix.exs` (runtime) and `client-elixir/Dockerfile` (escript) |

## Why

The reference Elixir protobuf implementation, pure Elixir, with the Google well-known
types (`Struct`, `Value`, `Timestamp`) that `signet-proto` imports shipped as modules. Its
`protoc-gen-elixir` plugin generates message modules and, with `plugins=grpc`, the
`GRPC.Service` and `GRPC.Stub` modules `grpc` calls through. The runtime and the generator
are pinned to one version so generated code and the library that loads it never drift.

Buf's schema registry hosts no Elixir or Erlang plugin, so `buf.gen.yaml` names the
escript as a `local:` plugin and the Docker toolchain image installs it with
`mix escript.install hex protobuf 0.17.1`. Generated code is not committed, as the Python,
Rust, and TypeScript clients also do; `scripts/test-client-elixir.sh` regenerates it
before every build.

The SPIFFE Workload API proto is vendored verbatim from go-spiffe under
`client-elixir/proto/` and generated with `package_prefix=spiffe.workload`, because it
declares no package and would otherwise land its modules at the top of the namespace.

## Alternatives considered

- `protobuf_generate` (a Mix task around the same plugin): adds a dependency to do what
  `buf generate` already does for the other clients.
