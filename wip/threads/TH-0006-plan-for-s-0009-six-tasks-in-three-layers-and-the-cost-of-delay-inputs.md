---
id: TH-0006
title: "Plan for S-0009: six tasks in three layers, and the cost of delay inputs"
anchor:
  path: wip/kanban/stories/S-0009-server-release-workflow-on-server-v-tags.md
  item: S-0009
status: resolved
participants: [planner-S-0009, alex]
created: 2026-10-27T22:27:04Z
updated: 2026-11-05T22:45:47Z
---

# TH-0006 Plan for S-0009: six tasks in three layers, and the cost of delay inputs

On wip/kanban/stories/S-0009-server-release-workflow-on-server-v-tags.md.

## Entries

### 2026-10-27T22:27:04Z planner-S-0009
Question first: S-0009 has no cost of delay inputs, and neither has E-0002, so `flai cod S-0009` refuses. I recommend time_lost_per_cycle 2h, with no revenue and no penalty: until this ships, every server release is cut by hand and the old v* tags cannot be reused. Alternatives: a penalty_per_week if a customer waits on the module path, or 0 if the release can wait for S-0008. Set the inputs with `flai edit S-0009 --time-lost-per-cycle 2h` (or on E-0002 to cover all its stories) and I will work out the value on the next run.

Plan (touches and the figures are in the story's Notes, under Planning):

Layer 1, no `after` between them, no shared files:
- T-0037 Rename the Go module to github.com/bytepunx/signet/server (go.mod, buf.gen.yaml, 56 Go files, the generated pb files).
- T-0039 Write release-server.yml (tests, binaries, images, chart, GitHub Release).
- T-0040 Write publish-server-main.yml (path-filtered main image).
- T-0041 install.sh: newest server/v* release.

Layer 2:
- T-0043 Document tag form, image tags, chart version, install.sh (waits for all of layer 1 so the text names what exists).

Layer 3:
- T-0045 Run the workflow on server/v0.0.0-test, check the outputs, delete the test artifacts (no files; waits for T-0043).

Story `after`: S-0007 only, for scripts/test-server.sh, which T-0039 calls.

Assumptions:
- A pre-release tag (a `-` suffix) pushes only the exact version image tag, no `latest` or `vX.Y`, and makes a pre-release GitHub Release, so the test tag cannot move `latest`. The story does not say so; I added it for safety.
- Generated .pb.go files embed go_package in their descriptors, so T-0037 regenerates them with `buf generate` against the BSR pin rather than sed. S-0020 may later change that pin; the two stories touch different lines of buf.gen.yaml.
- The release job runs scripts/test-server.sh before publishing, as ADR-0001's model says.
- Chart.yaml keeps placeholder version/appVersion; the workflow sets them at package time.
- T-0045 needs the DOCKERHUB_USERNAME and DOCKERHUB_TOKEN secrets from S-0008 and a push to the remote. It stops and asks if they are missing rather than waiting on S-0008 through `after`.
- Folder touch kept on the story: server/gen, because regeneration may rewrite files no task can name yet.
- The old server/.github workflows and release-please files stay for S-0006 to delete.

I propose no split, merge, or drop.

### 2026-10-29T17:54:10Z alex
accept

### 2026-11-05T22:45:47Z alex
Resolved: S-0009 was accepted
