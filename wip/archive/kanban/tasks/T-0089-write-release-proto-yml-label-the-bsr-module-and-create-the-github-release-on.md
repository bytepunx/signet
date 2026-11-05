---
id: T-0089
type: task
nature: feature
title: "write release-proto.yml: label the BSR module and create the GitHub Release on proto/v* tags"
status: done
parent: S-0010
owner: arobson
created: 2026-10-29T18:03:02Z
updated: 2026-10-29T18:03:43Z
transitions:
  - to: ready
    at: 2026-10-29T18:03:38Z
    by: agent-S-0010
  - to: in-progress
    at: 2026-10-29T18:03:38Z
    by: agent-S-0010
  - to: done
    at: 2026-10-29T18:03:43Z
    by: agent-S-0010
stream: S-0010
tags: []
touches: [".github/workflows/release-proto.yml"]
usage:
  source: log
  seconds: 5
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
# T-0089 write release-proto.yml: label the BSR module and create the GitHub Release on proto/v* tags

## Work

On proto/v* tags: validate the tag, buf push --label <version without prefix>, create a GitHub Release from the tag annotation.

## Done when

The workflow exists and meets criterion 2.

## Notes
