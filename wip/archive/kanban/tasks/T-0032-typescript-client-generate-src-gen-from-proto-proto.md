---
id: T-0032
type: task
nature: improvement
title: "TypeScript client: generate src/gen from ../proto/proto"
status: done
parent: S-0020
owner: alex
created: 2026-10-27T22:25:37Z
updated: 2026-11-06T08:41:12Z
transitions:
  - to: ready
    at: 2026-11-06T08:38:26Z
    by: agent-S-0020
  - to: in-progress
    at: 2026-11-06T08:38:27Z
    by: agent-S-0020
  - to: done
    at: 2026-11-06T08:41:12Z
    by: agent-S-0020
stream: S-0020
tags: [client-typescript, proto]
touches: [client-typescript/buf.gen.yaml, client-typescript/README.md]
usage:
  source: log
  seconds: 165
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 5
      output: 967
      cache_read: 174860
      cache_write: 6775
      cost: 0.0543
---
# T-0032 TypeScript client: generate src/gen from ../proto/proto

## Work

Replace the pinned BSR module input in `client-typescript/buf.gen.yaml` with `directory: ../proto/proto` and rewrite its comment, which describes the pin and the automation that bumped it. Run `buf generate` in `client-typescript/` and diff `src/gen` against what the `v0.4.0` pin produced. In `client-typescript/README.md` rewrite the opening paragraph and the regenerate section (about lines 4 and 26 to 32) so they say `buf generate` reads `../proto/proto` and no longer tell readers to bump a BSR pin. The echo example compiles against the committed `src/gen`, so its Dockerfile needs no change.

## Done when

- `client-typescript/buf.gen.yaml` names `directory: ../proto/proto` as its only input.
- `buf generate` in `client-typescript/` leaves `git diff client-typescript/src/gen` empty.
- `npm test` in `client-typescript/` passes.
- `client-typescript/README.md` no longer tells readers to bump a BSR pin.

## Notes
