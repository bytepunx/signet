---
id: T-0016
type: task
nature: feature
title: Signet.Restart facade with wait_for_restart debounce
status: done
parent: S-0002
owner: arobson
created: 2026-10-27T21:31:38Z
updated: 2026-10-27T22:07:46Z
transitions:
  - to: ready
    at: 2026-10-27T22:03:23Z
    by: agent-S-0002
  - to: in-progress
    at: 2026-10-27T22:03:23Z
    by: agent-S-0002
  - to: done
    at: 2026-10-27T22:07:46Z
    by: agent-S-0002
stream: S-0002
tags: []
touches: [client-elixir/lib/signet/restart.ex, client-elixir/test/signet/restart_test.exs]
after: [T-0014, T-0015]
usage:
  source: log
  seconds: 263
  estimated: true
  priced_by: harness
  models:
    - model: claude-fable-5-1
      input: 193
      output: 7253
      cache_read: 1235817
      cache_write: 25535
      cost: 1.1326
---

# T-0016 Signet.Restart facade with wait_for_restart debounce

## Work

- Add `Signet.Restart` with `watch_bundle/4`, `next_change/2`, `acknowledge/1`, `stop_watch/1`, `acquire_lock/5`, `release/1`, `token/1`, `expires_at/1`, and `wait_for_restart/6` (channel, namespace, service, ttl_ms, debounce_ms, opts): wait for the first change, then while changes keep arriving within `debounce_ms` wait again, then stop the watcher and acquire the lock.
- Tests in `test/signet/restart_test.exs` for the debounce and the acquire that follows, against the fakes.

## Done when

- Tests pass in the toolchain image with `--warnings-as-errors`.
- Criterion 3 of S-0002 is demonstrated by the tests.

## Notes
