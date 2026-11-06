---
id: T-0094
type: task
nature: remediation
title: Give the SDK container a passwd entry for the caller
status: done
parent: S-0031
owner: arobson
created: 2026-11-06T07:20:04Z
updated: 2026-11-06T07:20:35Z
transitions:
  - to: ready
    at: 2026-11-06T07:20:06Z
    by: agent-S-0031
  - to: in-progress
    at: 2026-11-06T07:20:06Z
    by: agent-S-0031
  - to: done
    at: 2026-11-06T07:20:35Z
    by: agent-S-0031
stream: S-0031
tags: []
touches: [scripts/test-client-csharp.sh]
usage:
  source: log
  seconds: 29
  estimated: true
  priced_by: estimate
  models:
    - model: claude-sonnet-5-5
      input: 12
      output: 62
      cache_read: 307500
      cache_write: 4602
      cost: 0.1114
---
# T-0094 Give the SDK container a passwd entry for the caller

## Work

Write a passwd and group file naming the caller's uid and gid (home /cache) under the cache directory, mount them read-only over /etc/passwd and /etc/group in `run()` in `scripts/test-client-csharp.sh`, keep `--user`. Test from an empty `.cache/dotnet` as a uid the image does not know.

## Done when

- The script passes from an empty `.cache/dotnet` as a uid with no image passwd entry.
- bin/, obj/, and the cache stay owned by the caller.

## Notes
