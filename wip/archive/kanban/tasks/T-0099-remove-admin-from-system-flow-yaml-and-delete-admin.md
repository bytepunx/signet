---
id: T-0099
type: task
nature: feature
title: Remove admin from system-flow.yaml and delete admin/
status: done
parent: S-0018
owner: arobson
created: 2026-11-06T08:14:39Z
updated: 2026-11-06T08:15:11Z
transitions:
  - to: ready
    at: 2026-11-06T08:15:04Z
    by: agent-S-0018
  - to: in-progress
    at: 2026-11-06T08:15:04Z
    by: agent-S-0018
  - to: done
    at: 2026-11-06T08:15:07Z
    by: agent-S-0018
stream: S-0018
tags: []
touches: [system-flow.yaml, admin/README.md, admin/CLAUDE.md, design/system/admin.md]
after: [T-0098]
usage:
  source: log
  seconds: 3
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 2
      output: 204
      cache_read: 58935
      cache_write: 1759
      cost: 0.015
---
# T-0099 Remove admin from system-flow.yaml and delete admin/

## Work

## Done when

## Notes
