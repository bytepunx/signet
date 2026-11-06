---
id: S-0029
type: story
nature: research
title: Verify publish-server-main.yml and install.sh after S-0009 reaches main
status: done
parent: E-0002
owner: alex
created: 2026-11-05T21:28:47Z
updated: 2026-11-06T07:49:36Z
transitions:
  - to: ready
    at: 2026-11-05T21:29:01Z
    by: agent
  - to: in-progress
    at: 2026-11-05T22:45:50Z
    by: agent-S-0029
  - to: review
    at: 2026-11-06T07:47:49Z
    by: agent-S-0029
  - to: done
    at: 2026-11-06T07:49:36Z
    by: alex
blocked:
  - from: 2026-11-05T22:46:15Z
    until: 2026-11-06T07:32:23Z
    reason: "waiting on operator (TH-0018): publish S-0009 to origin/main and cut a stable server/vX.Y.Z release"
  - from: 2026-11-06T07:37:55Z
    until: 2026-11-06T07:43:48Z
    reason: awaiting the operator on TH-0022
tags: [server]
topics: [release, github]
touches: [".github/workflows/publish-server-main.yml", install.sh, ".github/workflows/release-server.yml", design/system/release.md, docs/operators/release.md, design/issues/I-0011-flai-release-pending-tagged-server-v0-11-1-on-a-commit-that-is-on-no-branch-so.md, design/issues/summary.md, design/issues/I-0007-the-bsr-rate-limit-resource-exhausted-too-many-requests-fails-the-buf-generate.md, design/issues/I-0009-flai-check-finds-roles-definition-outside-the-story-at-close-out.md, design/issues/I-0010-flai-check-finds-item-blocked-closed-outside-the-story-at-close-out.md]
after: [S-0009]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
  config:
    effort: high
usage:
  source: log
  seconds: 684
  priced_by: harness
  turns:
    - day: 2026-11-05
      work: 15
    - day: 2026-11-06
      ceremony: 4
      hand_edits: 1
      work: 78
  models:
    - model: claude-sonnet-5-5
      input: 224
      output: 34409
      cache_read: 14344455
      cache_write: 276926
      cost: 2.8867
  strategic:
    - kind: planner
      seconds: 93
      estimated: true
      models:
        - model: claude-sonnet-5-5
          input: 26
          output: 11707
          cache_read: 717944
          cache_write: 75407
          cost: 0.4905
    - kind: orchestrator
      seconds: 99
      estimated: true
      models:
        - model: claude-sonnet-5-5
          input: 58
          output: 392
          cache_read: 1479320
          cache_write: 290591
          cost: 0.8204
forecast:
  duration: 1h
  delivery: 2026-11-06T10:00:00Z
  basis: "flai forecast gave 5m from the median of 65 s per size unit; adjusted to 1h because the work is waiting on CI runs (multi-arch image build, release run) and a two-platform install check; delivery assumes the operator answers TH-0019 within a few hours"
  by: planner-S-0029
  at: 2026-11-06T06:59:45Z
---
# S-0029 Verify publish-server-main.yml and install.sh after S-0009 reaches main

## Goal

Confirm on real GitHub runs the two behaviours of S-0009 that cannot be exercised before S-0009 is merged to `main` and a stable `server/v*` release exists: the `main` image is published only when `server/` changed, and `install.sh` installs the newest `server/v*` release. Fix either if it fails.

## Acceptance criteria
- [x] `publish-server-main.yml` pushes the `main` image only when `server/` changed: a push to `main` touching `server/` publishes the `main` tag, and a push touching only other paths does not run or publish it
- [x] `install.sh` installs the newest `server/v*` release on linux and darwin and ignores other components' releases

## Tasks
- T-0091 Confirm a stable server/v* release exists and its Release server run succeeded
- T-0092 Verify publish-server-main.yml publishes only when server/ changed
- T-0093 Verify install.sh installs the newest server/v* release on linux and darwin
- T-0095 Correct the release docs: the signetd image is tagged vX.Y.Z and vX.Y, not X.Y.Z

## Notes

Split from S-0009, whose criteria 2 and 4 could not be checked before merge: the only workflow run so far was a `server/v0.0.0-test` tag from a branch (a prerelease, since deleted), so neither the `main` path filter nor `install.sh` release resolution was exercised. This story waits on S-0009 (`after`) because it needs S-0009 merged and a stable `server/vX.Y.Z` release cut (S-0009 delivers the release workflow; cutting the release is the operator's call).

### Planning

Touches:
- `.github/workflows/publish-server-main.yml` and `install.sh`: declared.
- `.github/workflows/release-server.yml`: design. T-0091 checks the release run that `install.sh` depends on and fixes the workflow if that run fails; S-0009 wrote the file.
- `flai touches suggest` lists only files of the old layout (`Dockerfile`, `Makefile`, `cmd/signet/*`), from the 6 commits that changed the two declared files; none of them is a file this story changes, so none is added. No folder touch is kept.

`after: S-0009`: this story reads from S-0009 the files `.github/workflows/publish-server-main.yml`, `.github/workflows/release-server.yml`, and the `install.sh` release resolution, which S-0009 wrote and which exist on `main` only once S-0009 is published. S-0009 is done, so the dependency is met.

Tasks, in two layers:
- Layer 1: T-0091 confirms a stable `server/v*` release and a green Release server run; T-0092 verifies criterion 1. They share no path and no `after`, so they run together.
- Layer 2: T-0093 verifies criterion 2. It waits for T-0091 because `install.sh` reads the releases list.

Assumptions:
- The retag and the non-server push in TH-0019 are the operator's to allow; T-0091 and the negative half of T-0092 wait on that thread. The positive half of T-0092 needs no answer.
- The negative half may wait for the push that accepts this story, and then criterion 1 stays unticked at review.

Forecast: `flai forecast` gave 5m, the median of 65 s per unit of size over 9 done stories, times size 4. That counts only edits, and this story spends its time watching CI: a multi-arch image build and a release run take several minutes each, and the install check runs on two platforms. Adjusted to 1h of agent time. Delivery is set a few hours out on the assumption that the operator answers TH-0019 soon.

Cost of delay: `flai cod` has no inputs on this story or on E-0002, so no value is written; the inputs are asked for on the plan's thread.
