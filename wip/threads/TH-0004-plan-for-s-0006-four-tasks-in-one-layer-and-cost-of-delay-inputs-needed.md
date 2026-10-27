---
id: TH-0004
title: "Plan for S-0006: four tasks in one layer, and cost of delay inputs needed"
anchor:
  path: wip/kanban/stories/S-0006-baseline-component-tags-and-retire-release-please.md
  item: S-0006
status: open
participants: [planner-S-0006]
created: 2026-10-27T22:26:25Z
updated: 2026-10-27T22:26:25Z
---

# TH-0004 Plan for S-0006: four tasks in one layer, and cost of delay inputs needed

On wip/kanban/stories/S-0006-baseline-component-tags-and-retire-release-please.md.

## Entries

### 2026-10-27T22:26:25Z planner-S-0006
Recommendation first: set cost of delay inputs on S-0006 (or on E-0002 for all its stories) of time_lost_per_cycle 2h, revenue_per_week 0, penalty_per_week 0. Reason: while release-please files and client-erlang remain, the next `flai release --pending` would start six components at 0.0.0 and release a client that does not exist, and each cycle a person must fix that by hand. If you prefer another figure, give it; I did not set a value, since `flai cod S-0006` refuses without inputs, and I will write it once you have given them.

Plan: four tasks, all in one layer, with no `after` between them and no shared path, so they can run together.

- T-0022: delete `server/.github/` and the server's two release-please files.
- T-0023: delete `proto/.github/` and the proto module's two release-please files.
- T-0026: fold `client-erlang/README.md` into `client-elixir/README.md`, fix the `client-elixir/buf.gen.yaml` comment, delete `client-erlang/`, and run `flai subproject remove client-erlang`.
- T-0033: write the seven annotated `git tag -a` commands and one explicit `git push origin <tags>` in `docs/operators/release.md`, and in the narrative. Nothing is pushed.

Story figures: touches 16 declared plus `client-elixir/buf.gen.yaml` and `docs/operators/release.md` (no folder touch). Forecast 1h, flai's default with no history. `after: S-0005`, since `docs/operators/release.md` exists on main only once S-0005 is accepted; this holds the story in ready until then.

Assumptions:

- Criteria 1 and 2 need the operator to push the tags, so the story agent cannot tick them; they stay unticked with the commands ready.
- The criterion "no release-please file, manifest, or workflow" is read as files. Prose mentions of release-please in the clients' READMEs and `proto/README.md` belong to S-0010 to S-0016, which rewrite those READMEs.
- Deleting `server/.github/FUNDING.yml` loses the sponsor links, since GitHub reads only the root `.github/`. I did not add a task to move it to `.github/FUNDING.yml`; say so if you want one.
- The deleted CI, Publish, and Release workflows remain in git history for S-0007, S-0009, and S-0010 to read.

Proposed splits, merges, additions, or drops: none.
