---
id: S-0026
type: story
nature: remediation
title: item_new refuses an epic given touches with unrelated doc.topic findings
status: backlog
owner: alex
created: 2026-11-05T19:24:46Z
updated: 2026-11-05T19:24:46Z
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
    time_lost_per_cycle: 8m
    by: flai
    at: 2026-11-05T19:24:46Z
---
# S-0026 item_new refuses an epic given touches with unrelated doc.topic findings

## Goal

This story remediates [I-0001](../../../design/issues/I-0001-item-new-refuses-an-epic-given-touches-with-unrelated-doc-topic-findings.md), "item_new refuses an epic given touches with unrelated doc.topic findings". The issue recommends no solution yet: propose one from its instances before building it.

## Acceptance criteria
- [ ] The cause I-0001 describes no longer occurs, with a test that reproduces it where one fits
- [ ] I-0001 is closed with `flai issue close I-0001 --reason` saying what fixed it

## Tasks

## Notes

Cost of delay inputs set by flai from I-0001. time_lost_per_cycle 8m: 10m per occurrence × 1 occurrence ÷ 1.3 cycles of 168h (first reported 2026-10-27T19:48:36Z, 9 days before this story).
