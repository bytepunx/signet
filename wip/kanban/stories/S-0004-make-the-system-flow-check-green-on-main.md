---
id: S-0004
type: story
nature: remediation
title: Make the system-flow check green on main
status: review
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:49Z
updated: 2026-10-27T20:57:19Z
transitions:
  - to: ready
    at: 2026-10-27T19:47:51Z
    by: arobson
  - to: in-progress
    at: 2026-10-27T20:53:09Z
    by: agent-S-0004
  - to: review
    at: 2026-10-27T20:57:19Z
    by: agent-S-0004
tags: [server]
topics: [ci]
touches: [server/docs/secrets-workflow.md, design/system/overview.md]
agent:
  harness: claude-code
  model: claude-sonnet-5
usage:
  source: log
  seconds: 259
  priced_by: harness
  turns:
    - day: 2026-10-27
      ceremony: 5
      test_runs: 2
      work: 29
  models:
    - model: claude-sonnet-5
      input: 74
      output: 18786
      cache_read: 3155207
      cache_write: 109675
      cost: 1.2577
---
# S-0004 Make the system-flow check green on main

## Goal

The only CI workflow in the repository, `system-flow-check.yml`, passes on `main`. Its last run failed on markdownlint findings in `server/docs/secrets-workflow.md` (fenced code blocks without a language or blank lines around them), and `flai check --strict` warns on every run that `design/system/overview.md` has no `topics`.

## Acceptance criteria
- [x] `make lint-md` reports no findings for `server/docs/secrets-workflow.md`
- [x] `design/system/overview.md` carries `topics: [all]` and `flai check --strict` reports no warnings
- [ ] The system-flow check workflow run for the merge commit on `main` is green

## Tasks
- T-0005 Fix markdownlint findings and overview.md topics

## Notes

A patch to `server` on publish, as a documentation-only change (git.md).

Criterion 3 cannot be verified from this worktree: it depends on the `system-flow-check.yml` run for the merge commit that lands this change on `main`, which only runs after acceptance and merge. Both root causes (the markdownlint findings and the missing `topics`) are fixed and verified locally (`make lint-md` and `flai check --strict` both clean for the touched files); the operator should confirm the workflow run after merge.
