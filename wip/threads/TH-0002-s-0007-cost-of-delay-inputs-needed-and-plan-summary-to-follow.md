---
id: TH-0002
title: "S-0007: cost of delay inputs needed (and plan summary to follow)"
anchor:
  path: wip/kanban/stories/S-0007-root-ci-workflow-that-builds-and-tests-every-component-on-pull-requests.md
  item: S-0007
status: open
participants: [planner-S-0007]
created: 2026-10-27T22:24:37Z
updated: 2026-10-27T22:24:37Z
---

# TH-0002 S-0007: cost of delay inputs needed (and plan summary to follow)

On wip/kanban/stories/S-0007-root-ci-workflow-that-builds-and-tests-every-component-on-pull-requests.md.

## Entries

### 2026-10-27T22:24:37Z planner-S-0007
Recommended answer first: set no revenue, a modest penalty, and a time-lost input on S-0007 (or on E-0002 for all its stories): revenue_per_week 0, penalty_per_week 0, time_lost_per_cycle 4h. Reason: S-0007 has no revenue; its cost of delay is that S-0009 to S-0016 (every release workflow) wait on it and nothing is tested on pull requests until it lands, which costs about one manual test round per cycle.

`flai cod S-0007` refuses: neither S-0007 nor E-0002 has cost of delay inputs, and the inputs are yours. Set them with `flai edit S-0007 --revenue-per-week N --penalty-per-week N --time-lost-per-cycle 4h`, or answer here with the figures. I plan the tasks, touches and forecast meanwhile and write the cost of delay value once the inputs exist.
