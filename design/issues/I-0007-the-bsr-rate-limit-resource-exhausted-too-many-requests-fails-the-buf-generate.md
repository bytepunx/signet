---
id: I-0007
title: "The BSR rate limit (resource_exhausted: too many requests) fails the buf generate in the client test tiers during the close-out"
class: blocker
status: open
count: 2
cost: 25m
first_reported: 2026-10-29T18:07:14Z
last_reported: 2026-10-29T18:24:29Z
updated: 2026-10-29T18:24:29Z
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

## Remediation
