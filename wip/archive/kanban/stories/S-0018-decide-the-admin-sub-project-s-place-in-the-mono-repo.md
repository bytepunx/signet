---
id: S-0018
type: story
nature: research
title: Decide the admin sub-project's place in the mono-repo
status: done
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-11-06T08:17:44Z
transitions:
  - to: ready
    at: 2026-11-06T08:12:31Z
    by: alex
  - to: in-progress
    at: 2026-11-06T08:12:37Z
    by: agent-S-0018
  - to: review
    at: 2026-11-06T08:17:24Z
    by: agent-S-0018
  - to: done
    at: 2026-11-06T08:17:44Z
    by: alex
blocked:
  - from: 2026-11-06T08:12:51Z
    until: 2026-11-06T08:14:17Z
    reason: awaiting the operator on TH-0007
tags: [admin]
topics: [admin]
touches: [admin/README.md, admin/CLAUDE.md, system-flow.yaml, design/system/overview.md, design/system/admin.md]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
usage:
  source: log
  seconds: 203
  priced_by: harness
  turns:
    - day: 2026-11-06
      ceremony: 1
      work: 30
  models:
    - model: claude-sonnet-5-5
      input: 70
      output: 6635
      cache_read: 1912564
      cache_write: 57077
      cost: 0.4861
  strategic:
    - kind: planner
      seconds: 131
      estimated: true
      models:
        - model: claude-sonnet-5-5
          input: 14
          output: 1114
          cache_read: 278085
          cache_write: 37378
          cost: 0.1885
    - kind: orchestrator
      seconds: 25
      estimated: true
      models:
        - model: claude-sonnet-5-5
          input: 13
          output: 100
          cache_read: 328740
          cache_write: 66113
          cost: 0.2031
cost_of_delay:
  inputs:
    time_lost_per_cycle: 5m
    by: agent-S-0018
    at: 2026-11-06T08:14:33Z
forecast:
  duration: 45m
  delivery: 2026-11-06T14:56:00Z
  basis: "Its own forecast of 45m; 4th in the pull order with an in-progress limit of 2, behind S-0014, S-0020, S-0011 and S-0017."
  by: flai
  at: 2026-11-06T08:10:01Z
---
# S-0018 Decide the admin sub-project's place in the mono-repo

## Goal

`admin/` holds a README and nothing else but is registered as a SvelteKit project, so `flai check` warns about missing examples and a release could tag an empty component. The operator decides whether to scaffold the SvelteKit app now (as a story under its own epic) or remove `admin` from `system-flow.yaml` and keep the README as a design note under `design/system/` until the work starts.

## Acceptance criteria
- [x] The decision is recorded in `design/system/overview.md` and, if the project is removed, `system-flow.yaml` and the folder reflect it
- [x] `flai check --strict` reports nothing about `admin`

## Tasks
- T-0098 Write the admin design note and record the decision in the overview
- T-0099 Remove admin from system-flow.yaml and delete admin/

## Notes

### Planning

Touches, by source:

- `admin/README.md`, `system-flow.yaml`: declared.
- `admin/CLAUDE.md`: layout. It is the only other file in `admin/` and goes with the folder if the project is removed.
- `design/system/overview.md`: design. Criterion 1 names it as the place the decision is recorded.
- `design/system/admin.md`: design. The note that keeps the README's planned architecture if the project is removed.
- Co-change (`flai touches suggest`): no useful result, only `design/issues/summary.md` and `system-flow.lock.yaml`, which flai regenerates. Left out.
- No folder touch is kept.

Figures:

- Forecast: `flai forecast` gave 8m. Adjusted to 45m, since the work writes a note, edits the manifest, and deletes files, and the 8m median comes from smaller stories. Delivery follows from the pull order and moves with the decision.
- Cost of delay: no inputs on S-0018 or E-0002, so `flai cod` cannot give a value. They are the operator's; asked on the plan thread TH-0007 with a recommendation of `time_lost_per_cycle` 5m.
- The decision, remove or scaffold, is also the operator's and is asked on TH-0007. The tasks wait for it.
