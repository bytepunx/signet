---
id: S-0018
type: story
nature: research
title: Decide the admin sub-project's place in the mono-repo
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-11-05T19:24:46Z
transitions: []
tags: [admin]
topics: [admin]
touches: [admin/README.md, admin/CLAUDE.md, system-flow.yaml, design/system/overview.md, design/system/admin.md]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
usage:
  source: sum
  seconds: 0
  models: []
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
forecast:
  duration: 45m
  delivery: 2026-11-06T06:27:00Z
  basis: "Its own forecast of 45m; 9th in the pull order with an in-progress limit of 2, behind S-0020, S-0011, S-0012, S-0013, S-0014, S-0015, S-0016 and S-0017."
  by: flai
  at: 2026-11-05T19:24:46Z
---
# S-0018 Decide the admin sub-project's place in the mono-repo

## Goal

`admin/` holds a README and nothing else but is registered as a SvelteKit project, so `flai check` warns about missing examples and a release could tag an empty component. The operator decides whether to scaffold the SvelteKit app now (as a story under its own epic) or remove `admin` from `system-flow.yaml` and keep the README as a design note under `design/system/` until the work starts.

## Acceptance criteria
- [ ] The decision is recorded in `design/system/overview.md` and, if the project is removed, `system-flow.yaml` and the folder reflect it
- [ ] `flai check --strict` reports nothing about `admin`

## Tasks

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
