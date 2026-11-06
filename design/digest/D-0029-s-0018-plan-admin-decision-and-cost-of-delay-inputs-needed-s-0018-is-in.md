---
id: D-0029
title: "S-0018 plan: admin decision and cost of delay inputs needed: S-0018 is in progress, but this thread has no answer yet and the decision is yours"
kind: answer
sources: [TH-0007]
topics: [admin, release, ci, github, sveltekit, code, all]
paths: [admin/README.md, admin/CLAUDE.md, system-flow.yaml, design/system/overview.md, design/system/admin.md]
date: 2026-11-06
status: absorbed
absorbed_by: "design/system/overview.md#decisions"
asked_again: []
cited: []
---

# D-0029 S-0018 plan: admin decision and cost of delay inputs needed: S-0018 is in progress, but this thread has no answer yet and the decision is yours

S-0018 asked whether to remove or scaffold the `admin` sub-project, and what cost of delay to record.

The operator, alex, confirmed agent-S-0018's recommendation on TH-0007 at 2026-11-06T08:14:16Z: remove `admin` from `system-flow.yaml`, keep its architecture in `design/system/admin.md`, delete `admin/`, and record the decision in `design/system/overview.md`. Cost of delay: `time_lost_per_cycle` of 5m.

The reason: `admin/` held two files and no code, no story scheduled it, and a registered `sveltekit` component could be tagged and released empty (ADR-0001 retired `client-erlang` for the same reason). Scaffolding the SvelteKit app is a separate epic.

Source: TH-0007, S-0018.
