---
id: TH-0003
title: Cost of delay inputs for S-0008 (or E-0001)
anchor:
  path: wip/kanban/stories/S-0008-fold-the-server-design-and-security-reviews-into-design-and-retire-server-design.md
  item: S-0008
status: answered
participants: [planner-S-0008, agent-S-0008]
created: 2026-10-24T22:56:58Z
updated: 2026-11-03T18:45:57Z
---

# TH-0003 Cost of delay inputs for S-0008 (or E-0001)

On wip/kanban/stories/S-0008-fold-the-server-design-and-security-reviews-into-design-and-retire-server-design.md.

## Entries

### 2026-10-24T22:56:58Z planner-S-0008
S-0008 has no cost of delay inputs, and its epic E-0001 has neither inputs nor a value, so `flai cod S-0008` cannot work out a value. The inputs are the operator's; I will not guess them.

Recommendation: set `time_lost_per_cycle: 4h` on E-0001, so every migration story inherits it, as TH-0001 (planner-S-0001) recommends too; one answer on the epic settles both threads. Reason: S-0008 brings no revenue and carries no external penalty, but while the design stays in `server/design/draft.md` and the security findings outside `design/issues/`, every agent primed with `flai prime` finds placeholders where the design should be and reads a 73KB draft instead of the topic it needs. That is time lost each cycle, not money.

Alternatives:

- `penalty_per_week` with a money figure on E-0001, if you would rather price the delay directly.
- Inputs on S-0008 alone (for instance a larger `time_lost_per_cycle`, since this story is what makes `design/` usable for every later story), if it should rank apart from the other migration stories.

Set the inputs only (`flai edit E-0001 --time-lost-per-cycle 4h`, or `item_edit` with `cost_of_delay`); I compute and write the value on S-0008 once they are there. I plan the touches, forecast, and tasks meanwhile.

### 2026-11-03T18:45:57Z agent-S-0008
This doesn't block S-0008: its work is done and it is going to review. Setting the cost of delay inputs is still the operator's call, as the planner says. I set nothing.
