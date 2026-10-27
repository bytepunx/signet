---
id: S-0002
type: story
nature: feature
title: "Elixir client: coordinated restarts (bundle watch and restart lock)"
status: done
parent: E-0001
owner: alex
created: 2026-10-27T19:05:08Z
updated: 2026-10-27T22:25:06Z
transitions:
  - to: ready
    at: 2026-10-27T19:37:38Z
    by: alex
  - to: in-progress
    at: 2026-10-27T21:28:13Z
    by: agent-S-0002
  - to: review
    at: 2026-10-27T22:15:37Z
    by: agent-S-0002
  - to: done
    at: 2026-10-27T22:25:06Z
    by: alex
tags: []
topics: [elixir, client, restart-lock]
touches: [client-elixir/lib/signet/restart.ex, client-elixir/test/restart_test.exs, client-elixir/lib/signet/restart/streams.ex, client-elixir/test/support/fake_restart_streams.ex, client-elixir/lib/signet/restart/watcher.ex, client-elixir/test/signet/restart/watcher_test.exs, client-elixir/lib/signet/restart/lock.ex, client-elixir/test/signet/restart/lock_test.exs, client-elixir/test/signet/restart_test.exs, design/issues/I-0003-scripts-test-client-elixir-sh-stalls-in-docker-build-when-the-registry-is-unreachable-although-the-toolchain-image-exists-locally.md, design/issues/summary.md, client-elixir/CHANGELOG.md, client-elixir/README.md, client-elixir/examples/restart_on_change.exs, client-elixir/lib/signet.ex, docs/users/elixir-client.md]
agent:
  harness: claude-code
usage:
  source: log
  seconds: 2862
  priced_by: harness
  turns:
    - day: 2026-10-27
      ceremony: 1
      test_runs: 2
      hand_edits: 4
      work: 53
  models:
    - model: claude-fable-5-1
      input: 2720
      output: 102481
      cache_read: 17461493
      cache_write: 360800
      cost: 16.0026
---
# S-0002 Elixir client: coordinated restarts (bundle watch and restart lock)

## Goal

`Signet.Restart` mirrors the Go client's `WatchBundle`, `AcquireLock`, and `WaitForRestart`: a bundle-change watcher that reconnects with 1s-to-30s exponential backoff and coalesces rapid changes, a restart lock held by an open `AcquireRestartLock` stream with heartbeats at TTL/4 and a loss signal, and a convenience that waits for a debounced change and then acquires the lock.

## Acceptance criteria

- [x] `Signet.Restart.watch_bundle/4` delivers at most one pending change message per coalesced burst and reconnects with the Go client's backoff
- [x] `Signet.Restart.acquire_lock/5` blocks until ACQUIRED, heartbeats at TTL/4, exposes token and expiry, reports loss once, and releases idempotently by closing the stream
- [x] `Signet.Restart.wait_for_restart/6` debounces then acquires
- [x] Behaviour tests against fake streams cover all three, run by `make test`
- [x] README documents usage with a runnable example under `client-elixir/examples/`

## Tasks
- T-0013 Restart stream abstraction over SecretsService and its test fakes
- T-0014 Bundle watcher: coalesced change notifications with 1s-to-30s reconnect backoff
- T-0015 Restart lock: acquire, heartbeat at TTL/4, loss signal, idempotent release
- T-0016 Signet.Restart facade with wait_for_restart debounce
- T-0017 Document coordinated restarts: README, user docs, changelog, and a runnable example

## Notes

- Tests live under `client-elixir/test/signet/restart/` and `test/signet/restart_test.exs`, the `test/signet/` layout S-0001 chose, not the flat `test/restart_test.exs` the original touches named.
- Durations are milliseconds; the wire TTL is whole seconds (minimum 1) and heartbeats run at TTL/4 of the millisecond value. Go's capacity-one channel becomes one `{:bundle_changed, watcher}` message per burst until acknowledged; loss is one `{:lock_lost, lock, reason}` message.
- Criterion 4: `make test` runs `scripts/test.sh`, which runs the Elixir suite (67 tests, 29 of them the restart modules') in the Docker toolchain; `flai verify S-0002` ran it green. In an unattended session Docker Desktop's credential helper hangs every `docker build`; run with an empty `DOCKER_CONFIG` meanwhile (I-0003).
