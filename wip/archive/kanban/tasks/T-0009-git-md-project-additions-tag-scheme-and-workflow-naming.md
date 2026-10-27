---
id: T-0009
type: task
nature: research
title: "git.md project additions: tag scheme and workflow naming"
status: done
parent: S-0005
owner: arobson
created: 2026-10-27T21:23:45Z
updated: 2026-10-27T21:26:39Z
transitions:
  - to: ready
    at: 2026-10-27T21:24:56Z
    by: agent-S-0005
  - to: in-progress
    at: 2026-10-27T21:24:56Z
    by: agent-S-0005
  - to: done
    at: 2026-10-27T21:26:39Z
    by: agent-S-0005
stream: S-0005
tags: []
touches: [design/conventions/git.md]
after: [T-0006]
usage:
  source: log
  seconds: 103
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
# T-0009 git.md project additions: tag scheme and workflow naming

## Work

Under `## Project additions` in `design/conventions/git.md`, name the tag scheme (`<component>/vX.Y.Z`, component names from `system-flow.yaml`) and the workflow naming (`ci.yml`, `release-<component>.yml`), linking the ADR.

## Done when

The project additions name both and `flai check --strict` passes.

## Notes
