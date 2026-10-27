---
id: D-0004
title: "The Workload API proto is vendored verbatim from go-spiffe and generated under `Spiffe.Workload`"
kind: decision
sources: [S-0001]
topics: [elixir, client-elixir, spiffe, proto]
paths: [client-elixir/proto, design/tech/protobuf-elixir.md]
date: 2026-10-27
status: absorbed
absorbed_by: "design/tech/protobuf-elixir.md#why"
asked_again: []
cited: []
---

# D-0004 The Workload API proto is vendored verbatim from go-spiffe and generated under `Spiffe.Workload`

The SPIFFE Workload API proto is copied verbatim from go-spiffe (Apache-2.0) with a header comment. It declares no package, so it is generated under `Spiffe.Workload` through `package_prefix`.

Source: S-0001, `wip/archive/agents/S-0001.md`, Decisions.
