---
id: S-0031
type: story
nature: remediation
title: scripts/test-client-csharp.sh crashes in dotnet when the caller's uid has no passwd entry in the SDK image
status: in-progress
parent: E-0002
owner: alex
created: 2026-11-06T07:19:26Z
updated: 2026-11-06T07:22:42Z
transitions:
  - to: ready
    at: 2026-11-06T07:19:30Z
    by: agent
  - to: in-progress
    at: 2026-11-06T07:19:33Z
    by: agent
tags: [client-csharp]
topics: [github]
touches: [scripts/test-client-csharp.sh]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
  config:
    effort: high
usage:
  source: log
  seconds: 60
  estimated: true
  priced_by: estimate
  turns:
    - day: 2026-11-06
      ceremony: 1
      work: 12
  models:
    - model: claude-sonnet-5-5
      input: 30
      output: 173
      cache_read: 665849
      cache_write: 51486
      cost: 0.2561
---
# S-0031 scripts/test-client-csharp.sh crashes in dotnet when the caller's uid has no passwd entry in the SDK image

## Goal

`scripts/test-client-csharp.sh` passes from an empty `.cache/dotnet` on a host whose uid the SDK image does not know, so the `client-csharp` job in `ci.yml` is green. Today it fails on every full CI run on `main` (runs 42175371059 and 42158184314) before any test runs, with `System.IO.IOException: The system cannot open the device or file specified. : 'NuGet-Migrations'` from `MigrationRunner.Run` at `dotnet` first-run setup.

## Acceptance criteria
- [x] `scripts/test-client-csharp.sh` passes with `.cache/dotnet` removed first, as a uid with no entry in the image's `/etc/passwd`
- [ ] The `client-csharp` job in `ci.yml` is green on a run that exercises it
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
