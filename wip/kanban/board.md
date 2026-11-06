---
title: Board
updated: 2026-11-06
status: active
wip_limits:
  ready: 5
  in-progress: 2
  review: 3
order:
  - S-0020
  - S-0011
  - S-0012
  - S-0013
  - S-0014
  - S-0015
  - S-0016
  - S-0017
  - S-0018
  - S-0019
placed:
  S-0012:
    by: arobson
    at: 2026-10-27T19:47:51Z
  S-0013:
    by: arobson
    at: 2026-10-27T19:47:51Z
  S-0014:
    by: arobson
    at: 2026-10-27T19:47:51Z
  S-0015:
    by: arobson
    at: 2026-10-27T19:47:51Z
  S-0016:
    by: arobson
    at: 2026-10-27T19:47:51Z
  S-0017:
    by: arobson
    at: 2026-10-27T19:47:51Z
  S-0018:
    by: arobson
    at: 2026-10-27T19:47:51Z
  S-0019:
    by: arobson
    at: 2026-10-27T19:47:51Z
---

# Board

Columns: backlog, ready, in-progress, review, done. Limits apply to stories. `order` is the pull order: ready stories first, then backlog stories in refinement order. Change it with `flai order <story> --before|--after <other>`, `--top`, or `--bottom`, or by dragging a card within its column on the dashboard's board; stories it does not name come last, by ID.

Auto-accept: stories tagged `auto-accept` may be moved from review to done by an agent.
