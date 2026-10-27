---
id: T-0017
type: task
nature: feature
title: "Document coordinated restarts: README, user docs, changelog, and a runnable example"
status: done
parent: S-0002
owner: arobson
created: 2026-10-27T21:31:38Z
updated: 2026-10-27T22:10:35Z
transitions:
  - to: ready
    at: 2026-10-27T22:07:56Z
    by: agent-S-0002
  - to: in-progress
    at: 2026-10-27T22:07:56Z
    by: agent-S-0002
  - to: done
    at: 2026-10-27T22:10:35Z
    by: agent-S-0002
stream: S-0002
tags: []
touches: [client-elixir/README.md, client-elixir/examples/restart_on_change.exs, client-elixir/CHANGELOG.md, docs/users/elixir-client.md, client-elixir/lib/signet.ex]
after: [T-0016]
usage:
  source: log
  seconds: 159
  estimated: true
  priced_by: harness
  models:
    - model: claude-fable-5-1
      input: 209
      output: 7879
      cache_read: 1342467
      cache_write: 27739
      cost: 1.2303
---

# T-0017 Document coordinated restarts: README, user docs, changelog, and a runnable example

## Work

- Replace the "not yet ported" restart text in `client-elixir/README.md` with a "Coordinated restarts" section (usage, message protocol, TTL and heartbeat, loss), leaving a SOPS stub for S-0003.
- Add `client-elixir/examples/restart_on_change.exs`, the counterpart of `client-go/examples/restart-on-change`.
- Update `docs/users/elixir-client.md`, `client-elixir/CHANGELOG.md`, and the `Signet` moduledoc.

## Done when

- `mix compile --warnings-as-errors` and `mix format --check-formatted` pass in the toolchain image.
- Criterion 5 of S-0002 is met; markdown lint is clean.

## Notes
