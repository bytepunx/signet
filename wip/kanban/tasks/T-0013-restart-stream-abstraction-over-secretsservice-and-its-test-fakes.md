---
id: T-0013
type: task
nature: feature
title: Restart stream abstraction over SecretsService and its test fakes
status: done
parent: S-0002
owner: arobson
created: 2026-10-27T21:31:25Z
updated: 2026-10-27T21:38:12Z
transitions:
  - to: ready
    at: 2026-10-27T21:31:53Z
    by: agent-S-0002
  - to: in-progress
    at: 2026-10-27T21:31:53Z
    by: agent-S-0002
  - to: done
    at: 2026-10-27T21:38:12Z
    by: agent-S-0002
stream: S-0002
tags: []
touches: [client-elixir/lib/signet/restart/streams.ex, client-elixir/test/support/fake_restart_streams.ex]
usage:
  source: log
  seconds: 379
  estimated: true
  priced_by: harness
  models:
    - model: claude-fable-5-1
      input: 394
      output: 14828
      cache_read: 2526473
      cache_write: 52204
      cost: 2.3154
---

# T-0013 Restart stream abstraction over SecretsService and its test fakes

## Work

- Add `Signet.Restart.Streams`, a behaviour for the two SecretsService streams the restart module uses: `watch(channel, request)` returning an enumerable of `{:ok, resp} | {:error, reason}`, and the lock stream as `open_lock(channel)`, `send_lock(stream, request)`, `lock_responses(stream)`, `close_lock(stream)`.
- Add `Signet.Restart.Streams.GRPC` over `Signet.V1.SecretsService.Stub` (`watch_service_bundle`, `acquire_restart_lock`, `GRPC.Stub.send_request/2`, `recv/2`, `end_stream/1`), with `timeout: :infinity`.
- Add `Signet.FakeRestartStreams` under `test/support/` with scripted watch answers and a process-backed fake lock stream whose responses a test pushes after the fact and whose sends it records.

## Done when

- `mix compile --warnings-as-errors` passes in the toolchain image.
- The behaviour and the fake carry doc comments saying what they are for.

## Notes
