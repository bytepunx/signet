---
id: I-0004
title: "flai verify panics with a nil pointer dereference at verify.go:244 when its test tier is interrupted"
class: defect
status: open
count: 1
cost: 5m
first_reported: 2026-10-28T00:16:42Z
last_reported: 2026-10-28T00:16:42Z
updated: 2026-10-28T00:16:42Z
---

# I-0004 flai verify panics with a nil pointer dereference at verify.go:244 when its test tier is interrupted

## Description
flai verify panics with a nil pointer dereference at verify.go:244 when its test tier is interrupted

## Instances

### 2026-10-28T00:16:42Z
Story: S-0007.
The close-out was killed during the test tier after 20 minutes; flai verify S-0007 then printed 'stopped at test (interrupted)' followed by a SIGSEGV in verifyStory (flai/cmd/verify.go:244) instead of recording the stop.

## Remediation
