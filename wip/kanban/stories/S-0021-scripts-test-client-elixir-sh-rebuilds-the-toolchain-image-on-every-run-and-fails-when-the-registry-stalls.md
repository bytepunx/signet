---
id: S-0021
type: story
nature: improvement
title: scripts/test-client-elixir.sh rebuilds the toolchain image on every run and fails when the registry stalls
status: backlog
owner: alex
created: 2026-10-27T22:31:00Z
updated: 2026-10-27T22:54:19Z
transitions: []
tags: []
touches: [scripts/test-client-elixir.sh, scripts/integration-client-elixir.sh, scripts/ensure-client-elixir-image.sh, scripts/test-ensure-client-elixir-image.sh, scripts/test.sh, scripts/README.md, design/issues/I-0002-scripts-test-client-elixir-sh-rebuilds-the-toolchain-image-on-every-run-and-fails-when-the-registry-stalls.md, design/issues/summary.md]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
  config:
    effort: high
usage:
  source: sum
  seconds: 0
  models: []
  strategic:
    - kind: planner
      seconds: 84
      estimated: true
      models:
        - model: claude-sonnet-5-5
          input: 175
          output: 23424
          cache_read: 6498979
          cache_write: 319328
          cost: 2.7223
cost_of_delay:
  inputs:
    time_lost_per_cycle: 15m
    by: flai
    at: 2026-10-27T22:31:00Z
forecast:
  duration: 1h
  delivery: 2026-10-30T12:34:00Z
  basis: "Its own forecast of 1h; 16th in the pull order with an in-progress limit of 2, behind S-0006, S-0007, S-0008, S-0009, S-0010, S-0020, S-0011, S-0012, S-0013, S-0014, S-0015, S-0016, S-0017, S-0018 and S-0019."
  by: flai
  at: 2026-10-27T22:54:19Z
finalized:
  by: alex
  at: 2026-10-27T22:52:01Z
---
# S-0021 scripts/test-client-elixir.sh rebuilds the toolchain image on every run and fails when the registry stalls

## Goal

This story remediates [I-0002](../../../design/issues/I-0002-scripts-test-client-elixir-sh-rebuilds-the-toolchain-image-on-every-run-and-fails-when-the-registry-stalls.md), "scripts/test-client-elixir.sh rebuilds the toolchain image on every run and fails when the registry stalls". The issue recommends no solution yet: propose one from its instances before building it.

## Acceptance criteria
- [ ] The cause I-0002 describes no longer occurs, with a test that reproduces it where one fits
- [ ] I-0002 is closed with `flai issue close I-0002 --reason` saying what fixed it

## Tasks
- T-0073 Add the image-ensure script and its test
- T-0077 Use the ensure script in the Elixir test scripts and document it
- T-0083 Close I-0002 with the fix

## Notes

Cost of delay inputs set by flai from I-0002. time_lost_per_cycle 15m: 15m per occurrence × 1 occurrence ÷ 1 cycle of 168h (first reported 2026-10-27T21:58:04Z, 0 days before this story; under one cycle counts as one).

### Planning

Proposed solution: scripts/ensure-client-elixir-image.sh builds the toolchain image only when it is missing, its `signet.dockerfile-sha256` label differs from the hash of client-elixir/Dockerfile, or `SIGNET_REBUILD=1`. Both Elixir scripts call it. I-0003 (same cause, root cause the Docker Desktop credential helper) is fixed by the same change; closing it is the operator's choice.

Touches, none declared, all files, no folder touch kept:

- layout: scripts/test-client-elixir.sh and scripts/integration-client-elixir.sh both run `docker build` on every run; scripts/test.sh calls the first.
- design: the issue's remedy (skip the build when the image exists) gives the new scripts/ensure-client-elixir-image.sh and scripts/test-ensure-client-elixir-image.sh; scripts/README.md lists every script.
- co-change: `flai touches suggest` on the scripts listed scripts/README.md (2 of 214 commits); client-elixir/test/test_helper.exs was not taken because no Elixir code changes.
- layout: the issue file and design/issues/summary.md change when I-0002 is closed.

Forecast: flai forecast said 4m (median 116 s per unit of size, size 2, 0 touches). Adjusted to 1h: the new test, two script edits, and real Docker runs outweigh a size of 2.

Cost of delay: `flai cod` refuses until `planning.hour_rate` is set in system-flow.yaml, which is the operator's input. No value written; asked on the plan's thread.
