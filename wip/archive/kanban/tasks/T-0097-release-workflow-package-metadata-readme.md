---
id: T-0097
type: task
nature: feature
title: Release workflow, package metadata, README
status: done
parent: S-0013
owner: arobson
created: 2026-11-06T07:50:58Z
updated: 2026-11-06T07:51:02Z
transitions:
  - to: ready
    at: 2026-11-06T07:50:58Z
    by: agent-S-0013
  - to: in-progress
    at: 2026-11-06T07:50:58Z
    by: agent-S-0013
  - to: done
    at: 2026-11-06T07:51:02Z
    by: agent-S-0013
stream: S-0013
tags: []
touches: [".github/workflows/release-client-typescript.yml", client-typescript/package.json, client-typescript/README.md]
usage:
  source: log
  seconds: 4
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 2
      output: 254
      cache_read: 69697
      cache_write: 3202
      cost: 0.0223
---
# T-0097 Release workflow, package metadata, README

## Work

Add release-client-typescript.yml, fix package.json repository/bugs, document the tag form in the README.

## Done when

The three acceptance criteria hold.

## Notes
