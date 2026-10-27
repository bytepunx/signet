---
id: D-0009
title: "The watcher sends one `{:bundle_changed, watcher}` per burst until the owner acknowledges, where Go uses a channel of capacity 1"
kind: decision
sources: [S-0002]
topics: [elixir, client-elixir, restart-lock]
paths: [client-elixir/lib/signet/restart/watcher.ex, client-elixir/lib/signet/restart.ex, client-elixir/README.md]
date: 2026-10-27
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0009 The watcher sends one `{:bundle_changed, watcher}` per burst until the owner acknowledges, where Go uses a channel of capacity 1

Go's `<-chan struct{}` of capacity 1 becomes one message per burst in Elixir. The watcher sends it when a change arrives with none outstanding, and sends the next only after the owner acknowledges, with `Signet.Restart.next_change/2` or `Signet.Restart.acknowledge/1` from a `handle_info`. Durations are milliseconds, as Elixir timeouts are. The wire TTL is whole seconds, at least 1, and the heartbeat interval is TTL/4 of the millisecond value.

Source: S-0002, `wip/archive/agents/S-0002.md`, Decisions. Documented in `client-elixir/README.md`.
