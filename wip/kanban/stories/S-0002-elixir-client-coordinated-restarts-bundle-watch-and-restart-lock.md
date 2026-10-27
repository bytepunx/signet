---
id: S-0002
type: story
nature: feature
title: "Elixir client: coordinated restarts (bundle watch and restart lock)"
status: ready
parent: E-0001
owner: alex
created: 2026-10-27T19:05:08Z
updated: 2026-10-27T21:28:03Z
transitions:
  - to: ready
    at: 2026-10-27T19:37:38Z
    by: alex
tags: []
topics: [elixir, client, restart-lock]
touches: [client-elixir/lib/signet/restart.ex, client-elixir/test/restart_test.exs]
agent:
  harness: claude-code
---
# S-0002 Elixir client: coordinated restarts (bundle watch and restart lock)

## Goal

`Signet.Restart` mirrors the Go client's `WatchBundle`, `AcquireLock`, and `WaitForRestart`: a bundle-change watcher that reconnects with 1s-to-30s exponential backoff and coalesces rapid changes, a restart lock held by an open `AcquireRestartLock` stream with heartbeats at TTL/4 and a loss signal, and a convenience that waits for a debounced change and then acquires the lock.

## Acceptance criteria

- [ ] `Signet.Restart.watch_bundle/4` delivers at most one pending change message per coalesced burst and reconnects with the Go client's backoff
- [ ] `Signet.Restart.acquire_lock/5` blocks until ACQUIRED, heartbeats at TTL/4, exposes token and expiry, reports loss once, and releases idempotently by closing the stream
- [ ] `Signet.Restart.wait_for_restart/6` debounces then acquires
- [ ] Behaviour tests against fake streams cover all three, run by `make test`
- [ ] README documents usage with a runnable example under `client-elixir/examples/`

## Tasks

## Notes
