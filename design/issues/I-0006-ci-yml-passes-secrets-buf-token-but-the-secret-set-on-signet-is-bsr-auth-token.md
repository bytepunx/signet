---
id: I-0006
title: ci.yml passes secrets.BUF_TOKEN, but the secret set on signet is BSR_AUTH_TOKEN, so BSR calls run unauthenticated and hit the rate limit
class: defect
status: open
count: 1
cost: 10m
first_reported: 2026-10-29T17:59:55Z
last_reported: 2026-10-29T17:59:55Z
updated: 2026-10-29T17:59:55Z
---

# I-0006 ci.yml passes secrets.BUF_TOKEN, but the secret set on signet is BSR_AUTH_TOKEN, so BSR calls run unauthenticated and hit the rate limit

## Description
ci.yml passes secrets.BUF_TOKEN, but the secret set on signet is BSR_AUTH_TOKEN, so BSR calls run unauthenticated and hit the rate limit

## Instances

### 2026-10-29T17:59:55Z
Story: S-0009.
First occurrence.

## Remediation
