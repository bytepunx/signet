---
id: T-0033
type: task
nature: improvement
title: Write the baseline tag commands for the operator in the release runbook
status: done
parent: S-0006
owner: alex
created: 2026-10-27T22:25:38Z
updated: 2026-10-27T22:57:01Z
transitions:
  - to: ready
    at: 2026-10-27T22:56:55Z
    by: agent-S-0006
  - to: in-progress
    at: 2026-10-27T22:56:55Z
    by: agent-S-0006
  - to: done
    at: 2026-10-27T22:57:01Z
    by: agent-S-0006
stream: S-0006
tags: [server, proto]
touches: [docs/operators/release.md]
usage:
  source: log
  seconds: 6
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 1
      output: 129
      cache_read: 24953
      cache_write: 1815
      cost: 0.011
---
# T-0033 Write the baseline tag commands for the operator in the release runbook

## Work

Pushing tags is the operator's action (`safety.md`), so this task prepares the commands and pushes nothing.

For each of `server/v0.11.0`, `proto/v0.4.0`, `client-go/v0.7.0`, `client-python/v0.5.0`, `client-rust/v0.6.0`, `client-typescript/v0.9.0`, and `client-csharp/v0.6.0`, find the commit that imported that component with `git log` (the history carries `chore: import signet-proto . history into proto/` and the four client import commits, and `chore: move signet into server/ for the signet monorepo` for the server). Choose each commit so that `flai release --pending --dry-run` counts only work made after the import, and state in the runbook why that commit. Do not guess a hash: read it from `git log` and check it with `git cat-file -t`.

Add a "Baseline tags" section to `docs/operators/release.md` (written by S-0005) with the exact `git tag -a <tag> <commit> -m "<annotation>"` commands, each annotation naming the old repository's tag it continues (for example the server's root `v0.11.0`, proto's `v0.4.0`, and the clients' `go/v0.7.0`, `python-v0.5.0`, `rust-v0.6.0`, `typescript-v0.9.0`, `csharp-v0.6.0`), then one `git push origin <tag> ...` that names the seven tags and not `--tags`, which would push every tag. Say that the old root `v*` tags stay as history of the server.

Add what the operator should see afterwards: `flai release --pending --dry-run` plans `client-elixir` at its first version, plans nothing from `0.0.0` for any other component, and plans a patch of `server` and `proto` for the files this story deleted.

Put the same commands in the story's narrative with `flai stream log`.

The task waits for nothing in the story: it touches no file another task changes. The story waits for S-0005 because the runbook file does not exist on main until S-0005 is accepted.

## Done when

- `docs/operators/release.md` has a "Baseline tags" section with seven annotated-tag commands, each on a commit read from the history, and one push command naming the seven tags.
- Each annotation names the old repository's tag it continues.
- The section says what `flai release --pending --dry-run` shows once the tags are pushed.
- The narrative carries the same commands.
- `flai check --strict` and the markdown lint are clean.

## Notes

Layer 1. It shares no path with the other tasks of S-0006, so it runs together with them. Criteria 1 and 2 of the story cannot be met until the operator pushes the tags: the story agent leaves them unticked and says so in the story's notes.
