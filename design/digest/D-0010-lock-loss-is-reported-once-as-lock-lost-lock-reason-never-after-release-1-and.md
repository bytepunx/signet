---
id: D-0010
title: "Lock loss is reported once as `{:lock_lost, lock, reason}`, never after `release/1`, and the lock process outlives release"
kind: decision
sources: [S-0002]
topics: [elixir, client-elixir, restart-lock]
paths: [client-elixir/lib/signet/restart/lock.ex, client-elixir/README.md]
date: 2026-10-27
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0010 Lock loss is reported once as `{:lock_lost, lock, reason}`, never after `release/1`, and the lock process outlives release

Loss is reported once to the process that acquired the lock, and never after `release/1`. `release/1` half-closes the stream with `GRPC.Stub.end_stream/1`. The lock process stays alive, linked to its owner, so `token/1` and `expires_at/1` keep answering and a second `release/1` is a no-op.

Source: S-0002, `wip/archive/agents/S-0002.md`, Decisions. Documented in `client-elixir/README.md`.
