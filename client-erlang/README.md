# signet_client (Erlang)

Erlang client for [signet](https://github.com/bytepunx/signet), generated from
[bytepunx/signet-proto](https://github.com/bytepunx/signet-proto).

**Status:** not yet scaffolded. There is no maintained protobuf/gRPC remote plugin for
Erlang on the Buf Schema Registry, so this can't be wired up the way the Go, Python, Rust,
TypeScript, and C# clients are (a `buf.gen.yaml` pointing at a `remote:` plugin).

The [Elixir client](../client-elixir) settled the question for the BEAM: it runs a
**local** plugin (`protoc-gen-elixir`, from the `protobuf` hex package) inside the Docker
toolchain image it builds and tests in, with `buf generate` otherwise unchanged. An Erlang
client would take the same shape with [`gpb`](https://github.com/tomas-abrahamsson/gpb)'s
`protoc-gen-gpb` for messages, and either
[`grpcbox`](https://github.com/tsloughter/grpcbox) or the Elixir client's `grpc` package
(whose stubs are plain modules an Erlang application can call) for the service layer.
