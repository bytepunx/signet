---
id: T-0015
type: task
nature: improvement
title: Move the two security reviews to design/security with their history and index the folder
status: done
parent: S-0008
owner: alex
created: 2026-10-24T23:00:35Z
updated: 2026-11-03T18:33:14Z
transitions:
  - to: ready
    at: 2026-11-03T18:32:42Z
    by: agent-S-0008
  - to: in-progress
    at: 2026-11-03T18:32:42Z
    by: agent-S-0008
  - to: done
    at: 2026-11-03T18:33:14Z
    by: agent-S-0008
stream: S-0008
tags: [design, security, documentation]
touches: [server/design/security-findings.md, server/design/security-review-2026-08-31.md, design/security/README.md, design/security/security-findings.md, design/security/security-review-2026-08-31.md, design/README.md]
usage:
  source: log
  seconds: 32
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 4
      output: 1521
      cache_read: 275505
      cache_write: 5679
      cost: 0.1241
---
# T-0015 Move the two security reviews to design/security with their history and index the folder

## Work

Create `design/security/` with a `README.md` saying it holds the security reviews of signet, one file per review, dated, and that open findings are tracked in `design/issues/`. Move the two reviews whole with `git mv`, keeping their file names so history follows: `server/design/security-findings.md` (the 2026-07-01 audit) and `server/design/security-review-2026-08-31.md`. Give each front matter (`title`, `updated`, `status`, `topics` naming `server` and `security`) above its existing heading, and fix the relative links inside them that point at `design/draft.md` and `design/security-findings.md` so they resolve from the new location (the draft's sections become the `design/system/` topic files of the split task; link the folder's README where the exact file is not yet known). Add the folder to the index in `design/README.md` and a row to the layout table in `CLAUDE.md` saying what `design/security/` is for and that reviews there are records, edited only to mark a finding fixed. The reviews' `path:line` references to code stay as they are; the reviews say so themselves.

## Done when

- `design/security/README.md`, `design/security/security-findings.md`, and `design/security/security-review-2026-08-31.md` exist, the two reviews moved with `git mv` so `git log --follow` shows their history.
- `server/design/` holds only `draft.md`.
- `design/README.md` and `CLAUDE.md` name the folder.
- `make lint-md` and `flai check --strict` pass on the worktree.

## Notes

First layer; runs together with the draft split. `design/security/` is the planner's assumption for "an issues or security section" in criterion 3; the plan thread names it, and the operator may redirect it.
