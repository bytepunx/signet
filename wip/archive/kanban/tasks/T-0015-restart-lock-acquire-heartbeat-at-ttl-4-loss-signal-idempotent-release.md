---
id: T-0015
type: task
nature: feature
title: "Restart lock: acquire, heartbeat at TTL/4, loss signal, idempotent release"
status: done
parent: S-0002
owner: arobson
created: 2026-10-27T21:31:38Z
updated: 2026-10-27T22:03:20Z
transitions:
  - to: ready
    at: 2026-10-27T21:42:18Z
    by: agent-S-0002
  - to: in-progress
    at: 2026-10-27T21:42:18Z
    by: agent-S-0002
  - to: done
    at: 2026-10-27T22:03:20Z
    by: agent-S-0002
stream: S-0002
tags: []
touches: [client-elixir/lib/signet/restart/lock.ex, client-elixir/test/signet/restart/lock_test.exs]
after: [T-0013]
usage:
  source: log
  seconds: 1262
  estimated: true
  priced_by: harness
  models:
    - model: claude-fable-5-1
      input: 433
      output: 16326
      cache_read: 2781713
      cache_write: 57477
      cost: 2.5493
---

# T-0015 Restart lock: acquire, heartbeat at TTL/4, loss signal, idempotent release

## Work

- Add `Signet.Restart.Lock`, a `GenServer` started by `acquire/5` (channel, namespace, service, ttl_ms, opts) that opens the lock stream, sends the initial request with `ttl_seconds` (whole seconds, minimum 1), and blocks the caller until `MESSAGE_TYPE_ACQUIRED` or an error; `QUEUE_POSITION` messages are ignored.
- Heartbeats (`heartbeat: true`) every `ttl_ms / 4` while held; `TTL_EXTENDED` updates the expiry.
- `token/1`, `expires_at/1` (`DateTime`), `release/1` idempotent and closing the stream; loss reported once as `{:lock_lost, lock, reason}` to the acquiring process, never after release.
- Tests in `test/signet/restart/lock_test.exs` against the fake lock stream, mirroring `client-go/restart_test.go`.

## Done when

- Tests pass in the toolchain image with `--warnings-as-errors`.
- Criterion 2 of S-0002 is demonstrated by the tests.

## Notes
