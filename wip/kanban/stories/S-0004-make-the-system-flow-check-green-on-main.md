---
id: S-0004
type: story
nature: remediation
title: Make the system-flow check green on main
status: ready
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:49Z
updated: 2026-10-27T20:41:13Z
transitions:
  - to: ready
    at: 2026-10-27T19:47:51Z
    by: arobson
tags: [server]
topics: [ci]
touches: [server/docs/secrets-workflow.md, design/system/overview.md]
agent:
  harness: claude-code
  model: claude-sonnet-5
---
# S-0004 Make the system-flow check green on main

## Goal

The only CI workflow in the repository, `system-flow-check.yml`, passes on `main`. Its last run failed on markdownlint findings in `server/docs/secrets-workflow.md` (fenced code blocks without a language or blank lines around them), and `flai check --strict` warns on every run that `design/system/overview.md` has no `topics`.

## Acceptance criteria
- [ ] `make lint-md` reports no findings for `server/docs/secrets-workflow.md`
- [ ] `design/system/overview.md` carries `topics: [all]` and `flai check --strict` reports no warnings
- [ ] The system-flow check workflow run for the merge commit on `main` is green

## Tasks

## Notes

A patch to `server` on publish, as a documentation-only change (git.md).
