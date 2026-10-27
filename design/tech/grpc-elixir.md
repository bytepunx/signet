---
title: grpc and gun (Elixir gRPC client)
updated: 2026-10-27
status: active
topics: [client-elixir, grpc]
---

# grpc and gun

| | |
|---|---|
| Version | `grpc` 1.0.5 (with `grpc_core` 1.0.5), `gun` 2.4.1, `cowlib` 2.21.0 |
| Used in | `client-elixir/`: every connection to signet and to the SPIRE Workload API |
| Pinned in | `client-elixir/mix.exs` and `mix.lock` |

## Why

`grpc` (elixir-grpc) is the only maintained gRPC implementation for Elixir. Its 1.0
release split the package into a client (`grpc`) and a server (`grpc_server`); the client
is all this library needs. It takes a `GRPC.Credential` whose `ssl` keyword passes
straight through to Erlang `ssl`, which is what lets `Signet.Workload` hand it an
in-memory certificate chain, a PKCS#8 key, a trust bundle, and a `verify_fun`. Its
resolver accepts `unix:///path` targets, which the Workload API dial needs.

`gun` is the HTTP/2 transport `grpc` drives by default and is declared optional there;
it is pinned here because its adapter is the one that opens Unix domain sockets
(`:gun.open_unix/2`). `grpc` constrains it to `~> 2.4.0`.

## Known advisory

`mix hex.audit` reports CVE-2026-43966 (GHSA-w4f7-4cxr-rv3c) on `gun` and `cowlib`: HTTP
response splitting when structured header values are built from bytes outside the visible
ASCII range. This client builds every header it sends from fixed ASCII (`authorization:
Bearer` with a trimmed token, `workload.spiffe.io: true`) and never from untrusted input,
so the affected path is not reachable here. No `gun` release inside `grpc`'s constraint
carries a fix as of 2026-10-27; revisit when `grpc` lifts the constraint or `cowlib`
publishes one.

## Alternatives considered

- `grpc`'s Mint adapter: pure Elixir, but its connection process has no Unix-socket
  path, and the Workload API is only reachable over one.
- `grpcbox` (Erlang): server-oriented, with hand-written client dispatch over
  `gpb` messages; no generated typed stubs.
