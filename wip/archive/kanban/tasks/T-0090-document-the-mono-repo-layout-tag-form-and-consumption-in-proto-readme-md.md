---
id: T-0090
type: task
nature: feature
title: document the mono-repo layout, tag form, and consumption in proto/README.md
status: done
parent: S-0010
owner: arobson
created: 2026-10-29T18:03:02Z
updated: 2026-10-29T18:03:44Z
transitions:
  - to: ready
    at: 2026-10-29T18:03:38Z
    by: agent-S-0010
  - to: in-progress
    at: 2026-10-29T18:03:38Z
    by: agent-S-0010
  - to: done
    at: 2026-10-29T18:03:44Z
    by: agent-S-0010
stream: S-0010
tags: []
touches: [proto/README.md]
usage:
  source: log
  seconds: 6
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 1
      output: 179
      cache_read: 32306
      cache_write: 1907
      cost: 0.0127
---
# T-0090 document the mono-repo layout, tag form, and consumption in proto/README.md

## Work

Rewrite proto/README.md for the mono-repo: layout, proto/vX.Y.Z tags, in-repo consumers generate from the directory, external consumers pin the BSR label.

## Done when

README meets criterion 3.

## Notes
