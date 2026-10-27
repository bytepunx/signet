---
id: S-0021
type: story
nature: improvement
title: scripts/test-client-elixir.sh rebuilds the toolchain image on every run and fails when the registry stalls
status: backlog
owner: alex
created: 2026-10-27T22:31:00Z
updated: 2026-10-27T22:31:00Z
transitions: []
tags: []
agent:
  harness: claude-code
  model: claude-sonnet-5-5
  config:
    effort: high
draft: true
cost_of_delay:
  inputs:
    time_lost_per_cycle: 15m
    by: flai
    at: 2026-10-27T22:31:00Z
---
# S-0021 scripts/test-client-elixir.sh rebuilds the toolchain image on every run and fails when the registry stalls

## Goal

This story remediates [I-0002](../../../design/issues/I-0002-scripts-test-client-elixir-sh-rebuilds-the-toolchain-image-on-every-run-and-fails-when-the-registry-stalls.md), "scripts/test-client-elixir.sh rebuilds the toolchain image on every run and fails when the registry stalls". The issue recommends no solution yet: propose one from its instances before building it.

## Acceptance criteria
- [ ] The cause I-0002 describes no longer occurs, with a test that reproduces it where one fits
- [ ] I-0002 is closed with `flai issue close I-0002 --reason` saying what fixed it

## Tasks

## Notes

Cost of delay inputs set by flai from I-0002. time_lost_per_cycle 15m: 15m per occurrence × 1 occurrence ÷ 1 cycle of 168h (first reported 2026-10-27T21:58:04Z, 0 days before this story; under one cycle counts as one).
