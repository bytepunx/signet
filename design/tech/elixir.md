---
title: Elixir and Erlang/OTP
updated: 2026-10-27
status: active
topics: [client-elixir, elixir]
---

# Elixir and Erlang/OTP

| | |
|---|---|
| Version | Elixir 1.19.6 on Erlang/OTP 28.5.0.8 |
| Used in | `client-elixir/` |
| Image | `hexpm/elixir:1.19.6-erlang-28.5.0.8-debian-bookworm-20261020-slim`, pinned in `client-elixir/Dockerfile` |

## Why

The Elixir client exists so BEAM services can reach signet with the same SPIFFE mTLS and
bearer-token access the other five clients have. Erlang/OTP's `ssl` application is the
reason the SPIFFE path works at all without a third-party SPIFFE library: `verify_fun`
gives the client a hook on the peer certificate after chain validation, where it checks
the server's SPIFFE ID against the trust domain, and `server_name_indication: :disable`
turns off the hostname check an X.509-SVID can never pass. grpc-python has no such hook,
which is why the Python client's workload path does not work.

OTP 28 is the current major; nothing older is needed, and the toolchain is only ever run
inside the pinned image, never on a host, so there is no compatibility range to keep.

## Alternatives considered

- Installing Elixir on the host: refused by `design/conventions/safety.md`. The toolchain
  image and `scripts/test-client-elixir.sh` replace it.
- An Erlang (rebar3) client instead: the BEAM gRPC ecosystem's typed stubs come from
  `protoc-gen-elixir`; an Erlang application can call the Elixir client's modules.
