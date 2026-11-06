---
id: TH-0001
title: Cost of delay inputs for S-0001 (or E-0001)
anchor:
  path: wip/kanban/stories/S-0001-decide-proto-distribution-go-module-paths-and-the-release-model-for-the-mono-repo.md
  item: S-0001
status: open
participants: [planner-S-0001]
created: 2026-10-24T22:50:05Z
updated: 2026-10-24T22:50:05Z
---

# TH-0001 Cost of delay inputs for S-0001 (or E-0001)

On wip/kanban/stories/S-0001-decide-proto-distribution-go-module-paths-and-the-release-model-for-the-mono-repo.md.

## Entries

### 2026-10-24T22:50:05Z planner-S-0001
S-0001 has no cost of delay inputs, and its epic E-0001 has neither inputs nor a value, so `flai cod S-0001` cannot work out a value. The inputs are the operator's; I will not guess them.

Recommendation: set `time_lost_per_cycle: 4h` on E-0001, so every migration story inherits it. Reason: the migration brings no revenue and carries no external penalty, but each week the repositories stay half-merged, agents and contributors lose time to split generators, unresolvable Go module paths, missing CI, and placeholder docs. 4h a week is a conservative figure for that friction across nine sub-projects.

Alternatives:

- `penalty_per_week` with a money figure on E-0001, if you would rather price the delay directly.
- Inputs on S-0001 alone, if the decision story should rank apart from the other migration stories.

Set the inputs only (`flai edit E-0001 --time-lost-per-cycle 4h`, or `item_edit` with `cost_of_delay`); I will compute and write the value on S-0001 once they are there. I plan the touches, forecast, and tasks meanwhile.
