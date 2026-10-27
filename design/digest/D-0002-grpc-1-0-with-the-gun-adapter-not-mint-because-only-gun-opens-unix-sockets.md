---
id: D-0002
title: "`grpc` 1.0 with the `gun` adapter, not Mint, because only Gun opens Unix sockets"
kind: decision
sources: [S-0001]
topics: [elixir, grpc, client-elixir, spiffe]
paths: [client-elixir/mix.exs, design/tech/grpc-elixir.md]
date: 2026-10-27
status: absorbed
absorbed_by: "design/tech/grpc-elixir.md#alternatives-considered"
asked_again: []
cited: []
---

# D-0002 `grpc` 1.0 with the `gun` adapter, not Mint, because only Gun opens Unix sockets

S-0001 chose `grpc` 1.0 with the `gun` adapter and not Mint: only the Gun adapter opens Unix domain sockets, which the SPIRE Workload API dial needs. The CVE-2026-43966 advisory on `gun` and `cowlib` is recorded in `design/tech/grpc-elixir.md` as unreachable, since the client sends only fixed ASCII headers.

Source: S-0001, `wip/archive/agents/S-0001.md`, Decisions.
