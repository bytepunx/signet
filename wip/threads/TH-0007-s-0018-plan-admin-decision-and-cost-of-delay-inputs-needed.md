---
id: TH-0007
title: "S-0018 plan: admin decision and cost of delay inputs needed"
anchor:
  path: wip/kanban/stories/S-0018-decide-the-admin-sub-project-s-place-in-the-mono-repo.md
  item: S-0018
status: resolved
participants: [planner-S-0018, agent-S-0018, alex]
created: 2026-10-27T22:52:03Z
updated: 2026-11-06T08:15:15Z
---

# TH-0007 S-0018 plan: admin decision and cost of delay inputs needed

On wip/kanban/stories/S-0018-decide-the-admin-sub-project-s-place-in-the-mono-repo.md.

## Entries

### 2026-10-27T22:52:03Z planner-S-0018
Two inputs are yours; I will not guess past them.

1. Decision (recommended: remove). Remove `admin` from `system-flow.yaml`, move the README's architecture into a design note `design/system/admin.md`, delete `admin/` (README.md and CLAUDE.md), and record the decision in `design/system/overview.md`. Reason: `admin/` holds two files and no code, no story schedules it, and a registered `sveltekit` component can be tagged and released empty (ADR-0001 retired `client-erlang` for the same reason). The alternative is to scaffold the SvelteKit app now under its own epic, which is a larger story of its own that I would not fold into this one. Answer 'remove' or 'scaffold'.

2. Cost of delay inputs. S-0018 and E-0002 have none, so `flai cod` cannot work out a value. Recommended: `time_lost_per_cycle` 5m (the empty component only adds a warning and a release risk, no revenue and no penalty). Give revenue_per_week, penalty_per_week, or time_lost_per_cycle, or tell me to leave it unset.

Draft plan, assuming 'remove' (I will revise it when you answer; for 'scaffold' this story becomes the decision record only, and the scaffold goes to a new epic I would draft):
- T1 (layer 1) Write `design/system/admin.md` from `admin/README.md` and `admin/CLAUDE.md`, and record the decision in `design/system/overview.md`. Touches: design/system/admin.md, design/system/overview.md, admin/README.md, admin/CLAUDE.md.
- T2 (layer 2, after T1) Remove `admin` from `system-flow.yaml`, delete `admin/`, and run `flai check --strict`. Touches: system-flow.yaml, admin/README.md, admin/CLAUDE.md.

Assumptions: `flai check --strict` already reports nothing about `admin` on main, so criterion 2 holds after the change by not regressing. S-0006 also edits `system-flow.yaml` (drops `client-erlang`); the lines differ, so I set no `after` between the stories.

### 2026-11-06T08:12:49Z agent-S-0018 (recommendation)
S-0018 is in progress, but this thread has no answer yet and the decision is yours. I need two inputs before I write tasks:

1. Remove or scaffold `admin`? I recommend remove, as the planner did. That means moving `admin/README.md` and `admin/CLAUDE.md` into `design/system/admin.md`, recording the decision in `design/system/overview.md`, dropping `admin` from `system-flow.yaml` and deleting `admin/`.
2. Cost of delay: should I set `time_lost_per_cycle` to 5m, or leave it unset?

When you answer, I will write the two tasks and work them.

Source: design/system/overview.md

### 2026-11-06T08:14:16Z alex
Confirmed the recommendation of 2026-11-06T08:12:49Z agent-S-0018.

Source: design/system/overview.md

### 2026-11-06T08:15:15Z agent-S-0018
Resolved.
