---
id: T-0025
type: task
nature: improvement
title: "Script: protect main with the CI jobs as required checks"
status: done
parent: S-0008
owner: alex
created: 2026-10-27T22:25:22Z
updated: 2026-10-28T05:52:28Z
transitions:
  - to: ready
    at: 2026-10-28T05:52:05Z
    by: agent-S-0008
  - to: in-progress
    at: 2026-10-28T05:52:05Z
    by: agent-S-0008
  - to: done
    at: 2026-10-28T05:52:28Z
    by: agent-S-0008
stream: S-0008
tags: [release, github]
touches: [scripts/protect-main.sh]
usage:
  source: log
  seconds: 23
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 2
      output: 475
      cache_read: 70069
      cache_write: 3415
      cost: 0.0254
---
# T-0025 Script: protect main with the CI jobs as required checks

## Work

Write `scripts/protect-main.sh`, POSIX `sh` under `set -eu`. It sets branch protection on `main` of `bytepunx/signet` with `gh api -X PUT repos/<repo>/branches/main/protection`.

- The required status checks are the job names `.github/workflows/ci.yml` defines (`server`, `proto`, `client-go`, `client-python`, `client-typescript`, `client-rust`, `client-csharp`, `client-elixir`) and the system-flow check's `check` job. Read the names from `ci.yml` as S-0007 left it on main, and list them in one place at the top of the script.
- `strict` (branch up to date) is off, so a story branch rebased by `flai stream sync` need not be re-run.
- Administrators are not enforced (`enforce_admins: false`) and no pull request review is required, so the operator's `flai release --pending` push of accepted history still works. This follows the operator's answer on the plan thread; the script records the answer in a comment.
- The default is a dry run that prints the JSON body and the command; `--apply` runs it. `--show` prints the current protection with `gh api`.
- It prints what it does, runs from any directory, and fails clearly when `gh` is missing or not logged in.

It waits for no other task and shares no path with the other two scripts, so it runs in the first layer with them.

## Done when

- `scripts/protect-main.sh` exists and is executable; without `--apply` it changes nothing and prints the body and command.
- The body's contexts equal the job names in `.github/workflows/ci.yml` plus `check`.
- `flai check --strict` is clean.

## Notes

A required check that never reports blocks a pull request, so S-0007's path filters must skip at job level, not at workflow level; the runbook task records this.
