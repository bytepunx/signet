---
id: TH-0004
title: Cost of delay inputs for S-0004 (or E-0001)
anchor:
  path: wip/kanban/stories/S-0004-make-the-root-test-integration-and-smoke-targets-run-every-sub-project.md
  item: S-0004
status: open
participants: [planner-S-0004]
created: 2026-10-24T22:57:44Z
updated: 2026-10-24T22:57:44Z
---

# TH-0004 Cost of delay inputs for S-0004 (or E-0001)

On wip/kanban/stories/S-0004-make-the-root-test-integration-and-smoke-targets-run-every-sub-project.md.

## Entries

### 2026-10-24T22:57:44Z planner-S-0004
S-0004 has no cost of delay inputs, and its epic E-0001 has neither inputs nor a value, so `flai cod S-0004` cannot work out a value. The inputs are the operator's; I will not guess them.

Recommendation: answer TH-0001 by setting `time_lost_per_cycle: 4h` on E-0001, as planner-S-0001 recommended there. S-0004 then inherits it, and one answer settles both threads. Reason: `make test` runs nothing today, so every story's close-out verifies nothing, and every agent and contributor runs six toolchains by hand; that is lost time, not revenue or an external penalty.

Alternatives:

- `time_lost_per_cycle: 2h` on S-0004 alone, if the test entry points should rank below the decision and generation stories.
- `penalty_per_week` with a money figure on E-0001, if you would rather price the delay directly.

Set the inputs only (`flai edit E-0001 --time-lost-per-cycle 4h`, or `item_edit` with `cost_of_delay`); I will compute and write the value on S-0004 once they are there. Touches, forecast, and tasks are planned meanwhile.
