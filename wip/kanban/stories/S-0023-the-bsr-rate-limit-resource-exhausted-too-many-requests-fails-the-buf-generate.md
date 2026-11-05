---
id: S-0023
type: story
nature: remediation
title: "The BSR rate limit (resource_exhausted: too many requests) fails the buf generate in the client test tiers during the close-out"
status: backlog
owner: alex
created: 2026-11-05T19:24:45Z
updated: 2026-11-05T19:24:45Z
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
    time_lost_per_cycle: 50m
    by: flai
    at: 2026-11-05T19:24:45Z
---
# S-0023 The BSR rate limit (resource_exhausted: too many requests) fails the buf generate in the client test tiers during the close-out

## Goal

This story remediates [I-0007](../../../design/issues/I-0007-the-bsr-rate-limit-resource-exhausted-too-many-requests-fails-the-buf-generate.md), "The BSR rate limit (resource_exhausted: too many requests) fails the buf generate in the client test tiers during the close-out". The issue recommends no solution yet: propose one from its instances before building it.

## Acceptance criteria
- [ ] The cause I-0007 describes no longer occurs, with a test that reproduces it where one fits
- [ ] I-0007 is closed with `flai issue close I-0007 --reason` saying what fixed it

## Tasks

## Notes

Cost of delay inputs set by flai from I-0007. time_lost_per_cycle 50m: 25m per occurrence × 2 occurrences ÷ 1 cycle of 168h (first reported 2026-10-29T18:07:14Z, 7.1 days before this story).
