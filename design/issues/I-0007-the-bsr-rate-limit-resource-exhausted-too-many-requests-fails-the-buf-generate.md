---
id: I-0007
title: "The BSR rate limit (resource_exhausted: too many requests) fails the buf generate in the client test tiers during the close-out"
class: blocker
status: open
count: 1
cost: 10m
first_reported: 2026-10-29T18:07:14Z
last_reported: 2026-10-29T18:07:14Z
updated: 2026-10-29T18:07:14Z
---

# I-0007 The BSR rate limit (resource_exhausted: too many requests) fails the buf generate in the client test tiers during the close-out

## Description
The BSR rate limit (resource_exhausted: too many requests) fails the buf generate in the client test tiers during the close-out

## Instances

### 2026-10-29T18:07:14Z
Story: S-0010.
S-0010 close-out: test-client-csharp.sh, then five components, failed in buf generate with BSR resource_exhausted; a workflow-only change selects every component tier

## Remediation
