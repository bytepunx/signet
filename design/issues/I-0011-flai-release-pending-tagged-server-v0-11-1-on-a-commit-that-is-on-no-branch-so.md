---
id: I-0011
title: flai release --pending tagged server/v0.11.1 on a commit that is on no branch, so no release workflow ran
class: defect
status: open
count: 1
cost: 1h
first_reported: 2026-11-06T07:36:38Z
last_reported: 2026-11-06T07:36:38Z
updated: 2026-11-06T07:49:36Z
---

# I-0011 flai release --pending tagged server/v0.11.1 on a commit that is on no branch, so no release workflow ran

## Description
flai release --pending tagged server/v0.11.1 on a commit that is on no branch, so no release workflow ran

## Instances

### 2026-11-06T07:36:38Z
Story: S-0029.
S-0029: the operator's first publish pushed annotated tag server/v0.11.1 (dd8c982, 2026-11-05 18:52) pointing at 130a92b "chore: commit wip", a commit on no branch whose tree lacks .github/workflows/release-server.yml. GitHub started no Release server run, and no server/v* GitHub release existed, so install.sh and the story's two criteria could not be checked until the operator cut server/v0.11.2 on 2a21755 (run 42178552331, green). server/v0.11.1 is still on the remote. Cause not established here; the publish ran with a worktree or branch whose head was not main's. Check that flai release --pending tags only a commit reachable from the pushed branch, and that the tagged commit contains the workflow.

## Remediation

Story S-0033 remediates this issue, created from it at 2026-11-06T07:49:36Z.
