---
id: D-0008
title: "Elixir streaming I/O sits behind the `Signet.Restart.Streams` behaviour, with single-responsibility modules under `lib/signet/restart/`"
kind: decision
sources: [S-0002]
topics: [elixir, client-elixir, restart-lock, code]
paths: [client-elixir/lib/signet/restart, client-elixir/test/signet/restart, client-elixir/test/support/fake_restart_streams.ex]
date: 2026-10-27
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0008 Elixir streaming I/O sits behind the `Signet.Restart.Streams` behaviour, with single-responsibility modules under `lib/signet/restart/`

Streaming I/O goes behind the `Signet.Restart.Streams` behaviour, as `Signet.Spiffe.WorkloadAPI` does for the Workload API, so the watcher and lock are tested against `Signet.FakeRestartStreams` without gRPC (code-quality.md: no direct third-party I/O). Modules are `Signet.Restart` (facade), `.Watcher`, `.Lock`, and `.Streams`, under `lib/signet/restart/`, with tests under `test/signet/restart/`. That follows the `test/signet/` layout S-0001 chose, not the flat `test/restart_test.exs` the story's touches named.

Source: S-0002, `wip/archive/agents/S-0002.md`, Decisions.
