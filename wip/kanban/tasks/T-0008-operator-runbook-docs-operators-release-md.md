---
id: T-0008
type: task
nature: research
title: "Operator runbook: docs/operators/release.md"
status: done
parent: S-0005
owner: arobson
created: 2026-10-27T21:23:45Z
updated: 2026-10-27T21:26:33Z
transitions:
  - to: ready
    at: 2026-10-27T21:24:56Z
    by: agent-S-0005
  - to: in-progress
    at: 2026-10-27T21:24:56Z
    by: agent-S-0005
  - to: done
    at: 2026-10-27T21:26:33Z
    by: agent-S-0005
stream: S-0005
tags: []
touches: [docs/operators/release.md, docs/operators/index.md]
after: [T-0006]
usage:
  source: log
  seconds: 97
  estimated: true
  priced_by: harness
  models:
    - model: claude-fable-5-1
      input: 52
      output: 1400
      cache_read: 150640
      cache_write: 8360
      cost: 0.2754
---
# T-0008 Operator runbook: docs/operators/release.md

## Work

Write `docs/operators/release.md`: how to publish (`git fetch`, `flai release --pending`), what each component's release produces and where to find it, and the credentials and trusted publishers the workflows need. Link it from `docs/operators/index.md`.

## Done when

The runbook exists with front matter and the operators index links it.

## Notes
