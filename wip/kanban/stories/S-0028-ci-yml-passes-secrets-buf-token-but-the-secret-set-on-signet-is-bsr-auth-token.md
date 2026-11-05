---
id: S-0028
type: story
nature: remediation
title: ci.yml passes secrets.BUF_TOKEN, but the secret set on signet is BSR_AUTH_TOKEN, so BSR calls run unauthenticated and hit the rate limit
status: backlog
owner: arobson
created: 2026-11-05T20:45:12Z
updated: 2026-11-05T20:45:12Z
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
    time_lost_per_cycle: 10m
    by: flai
    at: 2026-11-05T20:45:12Z
---
# S-0028 ci.yml passes secrets.BUF_TOKEN, but the secret set on signet is BSR_AUTH_TOKEN, so BSR calls run unauthenticated and hit the rate limit

## Goal

This story remediates [I-0006](../../../design/issues/I-0006-ci-yml-passes-secrets-buf-token-but-the-secret-set-on-signet-is-bsr-auth-token.md), "ci.yml passes secrets.BUF_TOKEN, but the secret set on signet is BSR_AUTH_TOKEN, so BSR calls run unauthenticated and hit the rate limit". The issue recommends no solution yet: propose one from its instances before building it.

## Acceptance criteria
- [ ] The cause I-0006 describes no longer occurs, with a test that reproduces it where one fits
- [ ] I-0006 is closed with `flai issue close I-0006 --reason` saying what fixed it

## Tasks

## Notes

Cost of delay inputs set by flai from I-0006. time_lost_per_cycle 10m: 10m per occurrence × 1 occurrence ÷ 1 cycle of 168h (first reported 2026-10-29T17:59:55Z, 7.1 days before this story).
