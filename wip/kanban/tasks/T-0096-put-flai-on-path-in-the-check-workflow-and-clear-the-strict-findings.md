---
id: T-0096
type: task
nature: remediation
title: Put flai on PATH in the check workflow and clear the strict findings
status: done
parent: S-0032
owner: arobson
created: 2026-11-06T07:43:11Z
updated: 2026-11-06T07:43:20Z
transitions:
  - to: ready
    at: 2026-11-06T07:43:14Z
    by: agent-S-0032
  - to: in-progress
    at: 2026-11-06T07:43:14Z
    by: agent-S-0032
  - to: done
    at: 2026-11-06T07:43:20Z
    by: agent-S-0032
stream: S-0032
tags: []
touches: [".github/workflows/system-flow-check.yml", ".claude/agents/analyzer.md", ".claude/agents/curator.md", ".claude/agents/orchestrator.md", ".claude/agents/planner.md", wip/archive/kanban/stories/S-0010-proto-publish-workflow-bsr-push-from-main-and-label-on-proto-v-tags.md, wip/kanban/stories/S-0013-typescript-client-release-workflow-on-client-typescript-v-tags-with-npm-provenance.md]
usage:
  source: log
  seconds: 6
  estimated: true
  priced_by: estimate
  models:
    - model: claude-sonnet-5-5
      input: 2
      output: 43
      cache_read: 106872
      cache_write: 642
      cost: 0.0365
---
# T-0096 Put flai on PATH in the check workflow and clear the strict findings

## Work

Add `$(go env GOPATH)/bin` to `$GITHUB_PATH` after `go install` in `.github/workflows/system-flow-check.yml`. Run `flai role definition --write` and link the four built-in role definitions under `.claude/agents/`. Run `flai unblock S-0010`. Give S-0013 a reason for its `after` on S-0007 in its Notes.

## Done when

- The workflow puts the installed `flai` on `PATH`.
- `flai check --strict` has no finding but the ones `--strict` passes over.

## Notes
