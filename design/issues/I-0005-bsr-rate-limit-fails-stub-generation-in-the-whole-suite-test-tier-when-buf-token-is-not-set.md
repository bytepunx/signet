---
id: I-0005
title: BSR rate limit fails stub generation in the whole-suite test tier when BUF_TOKEN is not set
class: blocker
status: open
count: 1
cost: 30m
first_reported: 2026-10-28T05:55:16Z
last_reported: 2026-10-28T05:55:16Z
updated: 2026-10-28T05:55:16Z
---

# I-0005 BSR rate limit fails stub generation in the whole-suite test tier when BUF_TOKEN is not set

## Description
BSR rate limit fails stub generation in the whole-suite test tier when BUF_TOKEN is not set

## Instances

### 2026-10-28T05:55:16Z
Story: S-0008.
S-0008 close-out: scripts/test-client-go.sh and five other components fail with 'resource_exhausted: too many requests' from the BSR when generating stubs, unauthenticated

## Remediation
