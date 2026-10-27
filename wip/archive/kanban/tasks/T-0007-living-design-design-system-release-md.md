---
id: T-0007
type: task
nature: research
title: "Living design: design/system/release.md"
status: done
parent: S-0005
owner: arobson
created: 2026-10-27T21:23:45Z
updated: 2026-10-27T21:26:27Z
transitions:
  - to: ready
    at: 2026-10-27T21:24:56Z
    by: agent-S-0005
  - to: in-progress
    at: 2026-10-27T21:24:56Z
    by: agent-S-0005
  - to: done
    at: 2026-10-27T21:26:27Z
    by: agent-S-0005
stream: S-0005
tags: []
touches: [design/system/release.md, design/system/README.md]
after: [T-0006]
usage:
  source: log
  seconds: 91
  estimated: true
  priced_by: harness
  models:
    - model: claude-fable-5-1
      input: 34
      output: 917
      cache_read: 98669
      cache_write: 5476
      cost: 0.1804
---
# T-0007 Living design: design/system/release.md

## Work

Write `design/system/release.md`: the flow from acceptance to published artifact per component, naming each workflow file, trigger, artifact, registry, and credential, with a Mermaid diagram, linking the ADR. Add its row to `design/system/README.md`.

## Done when

The document exists with front matter, a Mermaid diagram, and one row per component, and the index lists it.

## Notes
