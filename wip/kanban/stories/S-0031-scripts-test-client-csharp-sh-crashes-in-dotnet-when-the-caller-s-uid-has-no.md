---
id: S-0031
type: story
nature: remediation
title: scripts/test-client-csharp.sh crashes in dotnet when the caller's uid has no passwd entry in the SDK image
status: review
parent: E-0002
owner: alex
created: 2026-11-06T07:19:26Z
updated: 2026-11-06T07:44:57Z
transitions:
  - to: ready
    at: 2026-11-06T07:19:30Z
    by: agent
  - to: in-progress
    at: 2026-11-06T07:19:33Z
    by: agent
  - to: review
    at: 2026-11-06T07:41:54Z
    by: agent-S-0031
blocked:
  - from: 2026-11-06T07:35:32Z
    until: 2026-11-06T07:36:38Z
    reason: awaiting the operator on TH-0021
  - from: 2026-11-06T07:37:00Z
    until: 2026-11-06T07:37:41Z
    reason: awaiting the operator on TH-0021
  - from: 2026-11-06T07:37:50Z
    until: 2026-11-06T07:39:34Z
    reason: awaiting the operator on TH-0021
tags: [client-csharp]
topics: [github]
touches: [scripts/test-client-csharp.sh, design/issues/I-0007-the-bsr-rate-limit-resource-exhausted-too-many-requests-fails-the-buf-generate.md, design/issues/summary.md, design/issues/I-0008-flai-check-finds-roles-baseline-outside-the-story-at-close-out.md, design/issues/I-0009-flai-check-finds-roles-definition-outside-the-story-at-close-out.md, design/issues/I-0010-flai-check-finds-item-blocked-closed-outside-the-story-at-close-out.md]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
  config:
    effort: high
usage:
  source: log
  seconds: 1123
  priced_by: harness
  empty_wakes: 1
  turns:
    - day: 2026-11-06
      ceremony: 4
      empty_wakes: 1
      work: 68
  models:
    - model: claude-sonnet-5-5
      input: 170
      output: 18572
      cache_read: 7176492
      cache_write: 122461
      cost: 1.3936
  strategic:
    - kind: orchestrator
      seconds: 37
      estimated: true
      models:
        - model: claude-sonnet-5-5
          input: 20
          output: 140
          cache_read: 504893
          cache_write: 87171
          cost: 0.2902
---
# S-0031 scripts/test-client-csharp.sh crashes in dotnet when the caller's uid has no passwd entry in the SDK image

## Goal

`scripts/test-client-csharp.sh` passes from an empty `.cache/dotnet` on a host whose uid the SDK image does not know, so the `client-csharp` job in `ci.yml` is green. Today it fails on every full CI run on `main` (runs 42175371059 and 42158184314) before any test runs, with `System.IO.IOException: The system cannot open the device or file specified. : 'NuGet-Migrations'` from `MigrationRunner.Run` at `dotnet` first-run setup.

## Acceptance criteria
- [x] `scripts/test-client-csharp.sh` passes with `.cache/dotnet` removed first, as a uid with no entry in the image's `/etc/passwd`
- [x] The `client-csharp` job in `ci.yml` is green on a run that exercises it
- [x] `bin/`, `obj/`, and the cache stay owned by the caller, as the script's header says

## Tasks
- T-0094 Give the SDK container a passwd entry for the caller

## Notes

Reproduced in `mcr.microsoft.com/dotnet/sdk:8.0` (SDK 8.0.427, digest `sha256:341aa6e8`), on linux/amd64 and linux/arm64. CI runs the container as uid 1001, which the image has no passwd entry for.

| Run as | Result |
|--------|--------|
| root | passes |
| uid 1654, the image's `app` user | passes |
| uid 1001 | crashes |
| uid 1001, gid 1654 | crashes |
| uid 1001, `TMPDIR` on the cache mount | crashes |
| uid 1001, `/tmp` as a fresh tmpfs | crashes |
| uid 1001, seccomp unconfined | crashes |
| uid 1001, `DOTNET_SKIP_FIRST_TIME_EXPERIENCE=1` | crashes |
| uid 1001, with a passwd file naming uid 1001 mounted on `/etc/passwd` | passes |

The cause is the missing passwd entry, not `/tmp` permissions: an strace as uid 1001 shows no attempt on `/tmp/.dotnet`. The fix gives the container a passwd entry for the caller and keeps `--user`, so files stay owned by the caller. A warm `.cache/dotnet` may hide this, because the crash is in first-run setup, so test from an empty cache.

Agent, S-0031:

- Criterion 2 is unticked: it needs a CI run on `main` after this branch is accepted and pushed, which cannot be seen here.
- Criteria 1 and 3 were verified on a Mac through Docker Desktop with `id` shimmed to uid 1001, from an empty `.cache/dotnet`, 56 of 56 tests passing. Files came out owned by the host user, uid 501; a Linux host with a real uid 1001 was not available.
- The fix writes `.cache/dotnet-ids/passwd` and `group` (gitignored under `.cache`) and mounts them read-only over `/etc/passwd` and `/etc/group`.
