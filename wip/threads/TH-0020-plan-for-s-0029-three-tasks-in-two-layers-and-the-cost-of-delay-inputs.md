---
id: TH-0020
title: "Plan for S-0029: three tasks in two layers, and the cost of delay inputs"
anchor:
  path: wip/kanban/stories/S-0029-verify-publish-server-main-yml-and-install-sh-after-s-0009-reaches-main.md
  item: S-0029
status: resolved
participants: [planner-S-0029, alex]
created: 2026-11-06T06:59:53Z
updated: 2026-11-06T07:32:18Z
---

# TH-0020 Plan for S-0029: three tasks in two layers, and the cost of delay inputs

On wip/kanban/stories/S-0029-verify-publish-server-main-yml-and-install-sh-after-s-0009-reaches-main.md.

## Entries

### 2026-11-06T06:59:53Z planner-S-0029
Plan: three research tasks in two layers, and one input I need from you.

Tasks:
- T-0091, layer 1: confirm a stable server/v* release exists and its Release server run is green. Touches .github/workflows/release-server.yml. It waits on your answer to TH-0019 (retag).
- T-0092, layer 1: verify criterion 1 on publish-server-main.yml. Touches .github/workflows/publish-server-main.yml. The positive half needs no answer; the negative half needs your go-ahead on TH-0019, or waits for the push that accepts this story.
- T-0093, layer 2, after T-0091: verify criterion 2, install.sh on darwin and on linux. Touches install.sh.

T-0091 and T-0092 share no path and no after, so they run together.

Assumptions:
- S-0029 touches the three files above; release-server.yml was added to the declared two, because T-0091 may fix it. No folder touch is kept.
- Forecast 1h of agent time, adjusted up from flai's 5m: the work waits on CI runs and a two-platform install check. Delivery is set to 2026-11-06T10:00:00Z and assumes TH-0019 is answered soon.
- `after: S-0009` is met, S-0009 being done. This story reads the three files it wrote.

Question, with my recommendation first:
- Cost of delay: I recommend writing none. This is a verification story with no revenue; it only closes S-0009's two unchecked criteria. If you want one, give revenue_per_week, penalty_per_week, or time_lost_per_cycle, on S-0029 or on E-0002. flai cod needs inputs and has none on either.

I am not splitting, merging, or dropping any task.

### 2026-11-06T07:32:18Z alex
Resolved.
