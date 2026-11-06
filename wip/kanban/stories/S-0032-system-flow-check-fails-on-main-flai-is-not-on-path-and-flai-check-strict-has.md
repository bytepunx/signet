---
id: S-0032
type: story
nature: remediation
title: "system-flow check fails on main: flai is not on PATH, and flai check --strict has findings behind it"
status: ready
parent: E-0002
owner: alex
created: 2026-11-06T07:19:26Z
updated: 2026-11-06T07:19:58Z
transitions:
  - to: ready
    at: 2026-11-06T07:19:58Z
    by: arobson
tags: []
topics: [github]
touches: [".github/workflows/system-flow-check.yml", ".claude/agents/analyzer.md", ".claude/agents/curator.md", ".claude/agents/orchestrator.md", ".claude/agents/planner.md"]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
  config:
    effort: high
---
# S-0032 system-flow check fails on main: flai is not on PATH, and flai check --strict has findings behind it

## Goal

The `system-flow check` workflow is green on `main`. Today it fails every push with `flai: command not found` (exit 127, run 42178552069): `go install` puts `flai` in `$(go env GOPATH)/bin`, which nothing adds to `PATH` on the runner. That hides a second failure, because `flai check --strict` also exits 1 on a fresh clone.

## Acceptance criteria
- [ ] `.github/workflows/system-flow-check.yml` puts the installed `flai` on `PATH`
- [ ] `flai check --strict` exits 0 on a fresh clone with flai v1.83.0, the version `@latest` installs in CI
- [ ] The `system-flow check` workflow is green on a pull request or on `main`
- [ ] I-0009 and I-0010 are closed with `flai issue close` saying what fixed them

## Tasks

## Notes

`flai check --strict` on a fresh clone reports 10 warnings. A trial in a scratch clone exits 0 with these three changes, and leaves only the four `roles.baseline` findings, which `--strict` passes over:

- `roles.definition` (4, I-0009): run `flai role definition --write`, then link `analyzer`, `curator`, `orchestrator`, and `planner` under `.claude/agents/` to `../../.flai-cache/agents/<role>.md`, as the finding says (ADR-0236). The links dangle in a fresh clone because `.flai-cache` is ignored, and the check accepts that.
- `item.blocked-closed` (1, I-0010): `flai unblock S-0010` closes the blocked interval left open on the archived story.
- `wip.after-reason` (1): S-0013 names S-0007 in `after` with no reason in its Notes.

`system-flow-check.yml` is identical to the template's copy in `system-flow`, so the `PATH` bug is upstream too, and this edit will show as a conflict at the next `flai upgrade`. `flai upgrade` is not part of this story: its dry run lists 13 conflicts, and it would not fix `PATH`. The `roles.baseline` findings (I-0008, S-0024) stay for it.
