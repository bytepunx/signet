---
title: Technology index
updated: 2026-10-27
status: active
---

# Technology choices

One file per technology: version, where used, why chosen, what was considered.

| Technology | Version | Used in | File |
|------------|---------|---------|------|
| Elixir and Erlang/OTP | 1.19.6 / 28.5.0.8 | `client-elixir/` | [elixir.md](elixir.md) |
| grpc and gun | 1.0.5 / 2.4.1 | `client-elixir/` | [grpc-elixir.md](grpc-elixir.md) |
| protobuf and protoc-gen-elixir | 0.17.1 | `client-elixir/` | [protobuf-elixir.md](protobuf-elixir.md) |
| buf | 1.71.0 | `proto/`, every client | [buf.md](buf.md) |
| age on OTP crypto | age format v1, no package | `client-elixir/` | [age-elixir.md](age-elixir.md) |
| sops (test binary) | 3.13.3 | `client-elixir/` toolchain image | [sops.md](sops.md) |
