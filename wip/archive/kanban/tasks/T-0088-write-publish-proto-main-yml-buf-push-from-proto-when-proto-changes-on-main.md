---
id: T-0088
type: task
nature: feature
title: "write publish-proto-main.yml: buf push from proto/ when proto/ changes on main"
status: done
parent: S-0010
owner: arobson
created: 2026-10-29T18:03:02Z
updated: 2026-10-29T18:03:38Z
transitions:
  - to: ready
    at: 2026-10-29T18:03:38Z
    by: agent-S-0010
  - to: in-progress
    at: 2026-10-29T18:03:38Z
    by: agent-S-0010
  - to: done
    at: 2026-10-29T18:03:38Z
    by: agent-S-0010
stream: S-0010
tags: []
touches: [".github/workflows/publish-proto-main.yml"]
usage:
  source: log
  seconds: 0
  models: []
---
# T-0088 write publish-proto-main.yml: buf push from proto/ when proto/ changes on main

## Work

Push to main with paths proto/** runs buf push from proto/ with BUF_TOKEN.

## Done when

The workflow exists and triggers only on proto/ changes on main.

## Notes
