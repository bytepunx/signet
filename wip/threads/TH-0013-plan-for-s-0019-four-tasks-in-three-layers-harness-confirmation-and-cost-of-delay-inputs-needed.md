---
id: TH-0013
title: "Plan for S-0019: four tasks in three layers, harness confirmation and cost of delay inputs needed"
anchor:
  path: wip/kanban/stories/S-0019-smoke-test-echo-images-built-from-the-clients-examples.md
  item: S-0019
status: open
participants: [planner-S-0019]
created: 2026-10-27T22:53:37Z
updated: 2026-10-27T22:53:37Z
---

# TH-0013 Plan for S-0019: four tasks in three layers, harness confirmation and cost of delay inputs needed

On wip/kanban/stories/S-0019-smoke-test-echo-images-built-from-the-clients-examples.md.

## Entries

### 2026-10-27T22:53:37Z planner-S-0019
Plan for S-0019 (smoke-test echo images).

Two inputs are yours, and I have not guessed past either.

1. Is the signet-smoke-test harness still in use? The story's Notes say it is built only if so. Recommended answer: yes, build it. If no, drop the story. Nothing in the repository references the harness, and it is not checked out beside this repository.
2. Cost of delay inputs. S-0019 and E-0002 have none, so `flai cod S-0019` refuses. Recommended: revenue_per_week 0, penalty_per_week 0, time_lost_per_cycle 0h. The images serve an external test harness, no revenue or penalty rides on them, and this is the least urgent story of E-0002. Give other figures if you have a reason. The value is written on my next run.

Tasks, all in the backlog:

| Layer | Task | Touches |
|-------|------|---------|
| 1 | T-0071 Script that builds the echo image of a client | scripts/build-echo-images.sh |
| 2 | T-0078 smoke-images.yml: build and push each client's echo image when its folder changes on main | .github/workflows/smoke-images.yml |
| 2 | T-0081 scripts/smoke.sh builds the echo images and runs or points at the harness | scripts/smoke.sh, scripts/README.md |
| 3 | T-0086 Operator guide for the smoke-test images and the harness | docs/operators/smoke-tests.md, docs/operators/index.md |

Layer 2 tasks share no path and run together. Story touches are the union of these files; no folder touch.

Assumptions:

- After: S-0020 only. It moves the python, rust, and csharp echo Dockerfiles' build context to the repository root, so the build script must read the Dockerfiles as S-0020 leaves them. S-0020 already follows S-0005, S-0007, and S-0011.
- The five languages are go, python, typescript, rust, and csharp, and the image names stay `ghcr.io/bytepunx/signet-echo-<language>` as the old workflow's.
- Elixir has no `examples/echo`. The build script refuses it and the workflow names where it is added, which meets the criterion's "once it has an echo example" clause. If you want an Elixir echo example, it is a separate story.
- The workflow uses `GITHUB_TOKEN` only (`packages: write`). The ghcr.io package visibility for the harness is an operator step, documented in T-0086; I did not add it to S-0008.
- `scripts/smoke.sh` builds the images and runs the harness only when `SIGNET_SMOKE_TEST_DIR` names a checkout, otherwise it prints what the harness needs. The harness's real entry point is unknown, so T-0081 reads its README rather than inventing one.
- Forecast 2h, raised from flai's 6m (which rests on three done stories of size 3). Delivery follows S-0020's forecast plus 2h.
- The operator guide is a new file, not a section of docs/operators/release.md, which S-0006, S-0007, and S-0008 edit.

I propose no split, merge, or drop.
