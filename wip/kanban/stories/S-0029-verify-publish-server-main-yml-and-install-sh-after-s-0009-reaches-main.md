---
id: S-0029
type: story
nature: research
title: Verify publish-server-main.yml and install.sh after S-0009 reaches main
status: in-progress
parent: E-0002
owner: alex
created: 2026-11-05T21:28:47Z
updated: 2026-11-05T22:46:15Z
transitions:
  - to: ready
    at: 2026-11-05T21:29:01Z
    by: agent
  - to: in-progress
    at: 2026-11-05T22:45:50Z
    by: agent-S-0029
blocked:
  - from: 2026-11-05T22:46:15Z
    reason: "waiting on operator (TH-0018): publish S-0009 to origin/main and cut a stable server/vX.Y.Z release"
tags: [server]
topics: [release, github]
touches: [".github/workflows/publish-server-main.yml", install.sh]
after: [S-0009]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
  config:
    effort: high
usage:
  source: log
  seconds: 57
  priced_by: harness
  turns:
    - day: 2026-11-05
      work: 15
  models:
    - model: claude-sonnet-5-5
      input: 42
      output: 6208
      cache_read: 1134522
      cache_write: 78426
      cost: 0.4893
  strategic:
    - kind: orchestrator
      seconds: 21
      estimated: true
      models:
        - model: claude-sonnet-5-5
          input: 18
          output: 104
          cache_read: 452760
          cache_write: 105984
          cost: 0.2426
---
# S-0029 Verify publish-server-main.yml and install.sh after S-0009 reaches main

## Goal

Confirm on real GitHub runs the two behaviours of S-0009 that cannot be exercised before S-0009 is merged to `main` and a stable `server/v*` release exists: the `main` image is published only when `server/` changed, and `install.sh` installs the newest `server/v*` release. Fix either if it fails.

## Acceptance criteria
- [ ] `publish-server-main.yml` pushes the `main` image only when `server/` changed: a push to `main` touching `server/` publishes the `main` tag, and a push touching only other paths does not run or publish it
- [ ] `install.sh` installs the newest `server/v*` release on linux and darwin and ignores other components' releases

## Tasks

## Notes

Split from S-0009, whose criteria 2 and 4 could not be checked before merge: the only workflow run so far was a `server/v0.0.0-test` tag from a branch (a prerelease, since deleted), so neither the `main` path filter nor `install.sh` release resolution was exercised. This story waits on S-0009 (`after`) because it needs S-0009 merged and a stable `server/vX.Y.Z` release cut (S-0009 delivers the release workflow; cutting the release is the operator's call).
