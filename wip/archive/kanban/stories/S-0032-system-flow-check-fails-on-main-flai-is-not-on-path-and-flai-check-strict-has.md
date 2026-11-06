---
id: S-0032
type: story
nature: remediation
title: "system-flow check fails on main: flai is not on PATH, and flai check --strict has findings behind it"
status: done
parent: E-0002
owner: alex
created: 2026-11-06T07:19:26Z
updated: 2026-11-06T07:49:45Z
transitions:
  - to: ready
    at: 2026-11-06T07:19:58Z
    by: arobson
  - to: in-progress
    at: 2026-11-06T07:42:03Z
    by: agent-S-0032
  - to: review
    at: 2026-11-06T07:48:52Z
    by: agent-S-0032
  - to: done
    at: 2026-11-06T07:49:45Z
    by: alex
tags: []
topics: [github]
touches: [".github/workflows/system-flow-check.yml", ".claude/agents/analyzer.md", ".claude/agents/curator.md", ".claude/agents/orchestrator.md", ".claude/agents/planner.md", design/issues/I-0009-flai-check-finds-roles-definition-outside-the-story-at-close-out.md, design/issues/summary.md, design/issues/I-0010-flai-check-finds-item-blocked-closed-outside-the-story-at-close-out.md, wip/archive/kanban/stories/S-0010-proto-publish-workflow-bsr-push-from-main-and-label-on-proto-v-tags.md]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
  config:
    effort: high
usage:
  source: log
  seconds: 419
  priced_by: harness
  turns:
    - day: 2026-11-06
      ceremony: 2
      hand_edits: 5
      work: 43
  models:
    - model: claude-sonnet-5-5
      input: 106
      output: 14649
      cache_read: 5792555
      cache_write: 124854
      cost: 1.2254
---
# S-0032 system-flow check fails on main: flai is not on PATH, and flai check --strict has findings behind it

## Goal

The `system-flow check` workflow is green on `main`. Today it fails every push with `flai: command not found` (exit 127, run 42178552069): `go install` puts `flai` in `$(go env GOPATH)/bin`, which nothing adds to `PATH` on the runner. That hides a second failure, because `flai check --strict` also exits 1 on a fresh clone.

## Acceptance criteria
- [x] `.github/workflows/system-flow-check.yml` puts the installed `flai` on `PATH`
- [x] `flai check --strict` exits 0 on a fresh clone with flai v1.83.0, the version `@latest` installs in CI
- [x] The `system-flow check` workflow is green on a pull request or on `main`
- [x] I-0009 and I-0010 are closed with `flai issue close` saying what fixed them

## Tasks
- T-0096 Put flai on PATH in the check workflow and clear the strict findings

## Notes

`flai check --strict` on a fresh clone reports 10 warnings. A trial in a scratch clone exits 0 with these three changes, and leaves only the four `roles.baseline` findings, which `--strict` passes over:

- `roles.definition` (4, I-0009): run `flai role definition --write`, then link `analyzer`, `curator`, `orchestrator`, and `planner` under `.claude/agents/` to `../../.flai-cache/agents/<role>.md`, as the finding says (ADR-0236). The links dangle in a fresh clone because `.flai-cache` is ignored, and the check accepts that.
- `item.blocked-closed` (1, I-0010): `flai unblock S-0010` closes the blocked interval left open on the archived story.
- `wip.after-reason` (1): S-0013 names S-0007 in `after` with no reason in its Notes.

`system-flow-check.yml` is identical to the template's copy in `system-flow`, so the `PATH` bug is upstream too, and this edit will show as a conflict at the next `flai upgrade`. `flai upgrade` is not part of this story: its dry run lists 13 conflicts, and it would not fix `PATH`. The `roles.baseline` findings (I-0008, S-0024) stay for it.

Criterion 3 needs a run on GitHub, which an agent here cannot see or start; the operator ticked it. Criterion 2 was verified on fresh clones of `story/S-0032` with flai built from `v1.83.0` and with 1.82.0: both exit 0 with no warnings. `flai unblock S-0010` wrote only to the main checkout, so the same `until:` line is committed on the branch too; both carry the same text.
