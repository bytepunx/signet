---
id: S-0027
type: story
nature: remediation
title: BSR rate limit fails stub generation in the whole-suite test tier when BUF_TOKEN is not set
status: backlog
owner: alex
created: 2026-11-05T19:25:57Z
updated: 2026-11-05T19:25:57Z
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
    time_lost_per_cycle: 25m
    by: flai
    at: 2026-11-05T19:25:57Z
---
# S-0027 BSR rate limit fails stub generation in the whole-suite test tier when BUF_TOKEN is not set

## Goal

This story remediates [I-0005](../../../design/issues/I-0005-bsr-rate-limit-fails-stub-generation-in-the-whole-suite-test-tier-when-buf-token-is-not-set.md), "BSR rate limit fails stub generation in the whole-suite test tier when BUF_TOKEN is not set". The issue recommends no solution yet: propose one from its instances before building it.

## Acceptance criteria
- [ ] The cause I-0005 describes no longer occurs, with a test that reproduces it where one fits
- [ ] I-0005 is closed with `flai issue close I-0005 --reason` saying what fixed it

## Tasks

## Notes

Cost of delay inputs set by flai from I-0005. time_lost_per_cycle 25m: 30m per occurrence × 1 occurrence ÷ 1.2 cycles of 168h (first reported 2026-10-28T05:55:16Z, 8.6 days before this story).
