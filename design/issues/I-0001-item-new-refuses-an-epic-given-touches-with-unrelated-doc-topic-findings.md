---
id: I-0001
title: item_new refuses an epic given touches with unrelated doc.topic findings
class: defect
status: open
count: 1
cost: 10m
first_reported: 2026-10-27T19:48:36Z
last_reported: 2026-10-27T19:48:36Z
updated: 2026-10-27T19:48:36Z
---

# I-0001 item_new refuses an epic given touches with unrelated doc.topic findings

## Description
item_new refuses an epic given touches with unrelated doc.topic findings

## Instances

### 2026-10-27T19:48:36Z
Calling the MCP item_new for an epic with a touches list was refused twice with three doc.topic findings naming design/tech/elixir.md, grpc-elixir.md, and protobuf-elixir.md ("topic client-elixir is used by nothing"), although client-elixir is a registered project and flai check --strict on the same tree was clean. The CLI flai epic new without touches succeeded, and flai edit then said the real rule: "E-0002 is an epic; touches belong to stories and tasks". The MCP refusal reported the wrong finding. Workaround: create epics without touches.

## Remediation
