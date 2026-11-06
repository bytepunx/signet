---
id: I-0007
title: "The BSR rate limit (resource_exhausted: too many requests) fails the buf generate in the client test tiers during the close-out"
class: blocker
status: open
count: 4
cost: 21m
first_reported: 2026-10-29T18:07:14Z
last_reported: 2026-11-06T07:37:40Z
updated: 2026-11-06T07:37:40Z
---

# I-0007 The BSR rate limit (resource_exhausted: too many requests) fails the buf generate in the client test tiers during the close-out

## Description
The BSR rate limit (resource_exhausted: too many requests) fails the buf generate in the client test tiers during the close-out

## Instances

### 2026-10-29T18:07:14Z
Story: S-0010.
S-0010 close-out: test-client-csharp.sh, then five components, failed in buf generate with BSR resource_exhausted; a workflow-only change selects every component tier

### 2026-10-29T18:24:29Z
Story: S-0010.
Retried the close-out four times over about 30 minutes; every run failed in buf generate with resource_exhausted. BUF_TOKEN is unset in the agent environment.

### 2026-11-06T07:30:28Z
Story: S-0031.
S-0031: two close-outs stopped at the test tier on resource_exhausted from buf generate (client-csharp, client-rust, then server and clients)

### 2026-11-06T07:37:40Z
Story: S-0029.
S-0029: the close-out stopped at the test tier at f2b2e47 (job S-0029-20261106T073646Z, 40s) on a docs-only change. test.sh failed server, client-go, client-python, client-typescript, client-rust and client-csharp; client-elixir passed. Running scripts/test-client-go.sh in the worktree shows `Failure: resource_exhausted: too many requests` from buf generate. The same cause as S-0031's TH-0021. The change is not the cause.

## Remediation

Story S-0023 remediates this issue, created from it at 2026-11-05T19:24:45Z.
