---
id: TH-0008
title: "S-0021 plan: skip the toolchain image build when current; hour rate needed"
anchor:
  path: wip/kanban/stories/S-0021-scripts-test-client-elixir-sh-rebuilds-the-toolchain-image-on-every-run-and-fails-when-the-registry-stalls.md
  item: S-0021
status: resolved
participants: [planner-S-0021, alex]
created: 2026-10-27T22:52:34Z
updated: 2026-11-06T08:10:00Z
---

# TH-0008 S-0021 plan: skip the toolchain image build when current; hour rate needed

On wip/kanban/stories/S-0021-scripts-test-client-elixir-sh-rebuilds-the-toolchain-image-on-every-run-and-fails-when-the-registry-stalls.md.

## Entries

### 2026-10-27T22:52:34Z planner-S-0021
Plan for S-0021 (remediates I-0002; I-0003 shares the cause and the same fix).

Question for you first: `flai cod S-0021` refuses because `planning.hour_rate` is unset in system-flow.yaml. Recommended answer: set it to what an hour of your work costs in USD (e.g. 100), then `flai cod S-0021` gives the value. I have not written a cost of delay value and will not guess one. Meanwhile the story's time_lost_per_cycle input (15m) stands.

Approach: scripts/ensure-client-elixir-image.sh builds signet-client-elixir:toolchain only when the image is missing or its `signet.dockerfile-sha256` label differs from the hash of client-elixir/Dockerfile (SIGNET_REBUILD=1 forces). `docker image inspect` talks only to the local daemon, so it does not hit the registry or the credential helper. test-client-elixir.sh and integration-client-elixir.sh both call it instead of running `docker build` on every run.

Tasks, in layers:
- Layer 1: T-A "Add the image-ensure script and its test" (scripts/ensure-client-elixir-image.sh, scripts/test-ensure-client-elixir-image.sh; the test puts a fake `docker` on PATH and asserts the build is skipped when the label matches, run when the image is missing, run when the Dockerfile hash differs, and run when SIGNET_REBUILD=1).
- Layer 2: T-B "Use the script in the test scripts and document it" (scripts/test-client-elixir.sh, scripts/integration-client-elixir.sh, scripts/test.sh runs the new test, scripts/README.md). After T-A.
- Layer 3: T-C "Close I-0002 with the fix" (design/issues/I-0002 file, design/issues/summary.md). After T-B. I-0003 needs its own `flai issue close`; I suggest linking it to this story or closing it with the same reason at acceptance.

Assumptions: only Docker, not a registry timeout, is the cost, so no --pull=false (the base image is pulled only when a rebuild is needed); POSIX sh with shasum or sha256sum; no change to client-elixir/Dockerfile. Forecast and touches are in the story's Notes under Planning.

### 2026-10-27T22:53:18Z planner-S-0021
Tasks are created: T-0073 (layer 1), T-0077 (layer 2, after T-0073), T-0083 (layer 3, after T-0077); these are the T-A, T-B, T-C above. Touches and forecast (1h, delivery 2026-10-28T13:00:00Z) are written on S-0021. Still open: `planning.hour_rate` for the cost of delay value.

### 2026-11-06T08:10:00Z alex
Resolved: S-0021 was accepted
