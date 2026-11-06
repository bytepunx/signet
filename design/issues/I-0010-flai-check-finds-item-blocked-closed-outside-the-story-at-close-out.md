---
id: I-0010
title: "flai check finds `item.blocked-closed` outside the story at close-out"
class: efficiency
status: closed
count: 3
first_reported: 2026-11-05T20:46:48Z
last_reported: 2026-11-06T07:37:25Z
updated: 2026-11-06T07:43:48Z
---

# I-0010 flai check finds `item.blocked-closed` outside the story at close-out

## Description
flai check finds `item.blocked-closed` outside the story at close-out

## Instances

### 2026-11-05T20:46:48Z
Story: S-0009.
flai check found outside the story:
`wip/archive/kanban/stories/S-0010-proto-publish-workflow-bsr-push-from-main-and-label-on-proto-v-tags.md`: done item still has an open blocked interval

### 2026-11-06T07:23:54Z
Story: S-0031.
flai check found outside the story:
`wip/archive/kanban/stories/S-0010-proto-publish-workflow-bsr-push-from-main-and-label-on-proto-v-tags.md`: done item still has an open blocked interval

### 2026-11-06T07:37:25Z
Story: S-0029.
flai check found outside the story:
`wip/archive/kanban/stories/S-0010-proto-publish-workflow-bsr-push-from-main-and-label-on-proto-v-tags.md`: done item still has an open blocked interval

## Remediation

Story S-0030 remediates this issue, created from it at 2026-11-05T22:45:47Z.
Closed 2026-11-06T07:43:48Z: fixed by S-0032: flai unblock S-0010 closed the blocked interval left open on the archived story
