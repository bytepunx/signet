---
id: T-0014
type: task
nature: feature
title: "Bundle watcher: coalesced change notifications with 1s-to-30s reconnect backoff"
status: done
parent: S-0002
owner: arobson
created: 2026-10-27T21:31:38Z
updated: 2026-10-27T21:42:15Z
transitions:
  - to: ready
    at: 2026-10-27T21:38:20Z
    by: agent-S-0002
  - to: in-progress
    at: 2026-10-27T21:38:20Z
    by: agent-S-0002
  - to: done
    at: 2026-10-27T21:42:15Z
    by: agent-S-0002
stream: S-0002
tags: []
touches: [client-elixir/lib/signet/restart/watcher.ex, client-elixir/test/signet/restart/watcher_test.exs]
after: [T-0013]
usage:
  source: log
  seconds: 235
  estimated: true
  priced_by: harness
  models:
    - model: claude-fable-5-1
      input: 193
      output: 7288
      cache_read: 1241825
      cache_write: 25659
      cost: 1.1381
---

# T-0014 Bundle watcher: coalesced change notifications with 1s-to-30s reconnect backoff

## Work

- Add `Signet.Restart.Watcher`, a `GenServer` that owns a reader process running the Go `watchLoop`: open the watch stream, forward `EVENT_TYPE_CHANGED`, on end or error sleep with backoff 1s doubling to 30s (reset on any received event) and reopen.
- One `{:bundle_changed, watcher}` message to the owner per burst: sent when a change arrives with none outstanding; the next only after `acknowledge/1`.
- `next_change/2` receives then acknowledges, returning `:ok` or `:timeout`; `stop/1` ends the watcher and its stream. Backoff sleep injectable (`:sleep`) for tests.
- Tests in `test/signet/restart/watcher_test.exs` against `Signet.FakeRestartStreams`: coalescing, reconnect with the recorded backoff schedule, reset after an event, and stop.

## Done when

- Tests pass in the toolchain image with `--warnings-as-errors`.
- Criterion 1 of S-0002 is demonstrated by the tests.

## Notes
