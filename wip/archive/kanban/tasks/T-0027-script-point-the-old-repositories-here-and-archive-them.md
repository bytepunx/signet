---
id: T-0027
type: task
nature: improvement
title: "Script: point the old repositories here and archive them"
status: done
parent: S-0008
owner: alex
created: 2026-10-27T22:25:25Z
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
touches: [scripts/archive-old-repos.sh]
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
# T-0027 Script: point the old repositories here and archive them

## Work

Write `scripts/archive-old-repos.sh`, POSIX `sh` under `set -eu`. For each of `bytepunx/signet-clients`, `bytepunx/signet-proto`, and `bytepunx/signet-admin`:

1. Replace its `README.md` with a short pointer written by a heredoc in the script: the repository is archived, its code lives under the matching folders of `https://github.com/bytepunx/signet` (`client-*`, `proto`, `admin`), and releases now come from there as `<component>/vX.Y.Z` tags. Read the file's current `sha` with `gh api repos/<repo>/contents/README.md --jq .sha` and update it with `gh api -X PUT`, the content base64-encoded.
2. Archive it with `gh repo archive <repo> --yes`.

The default is a dry run that prints each step; `--apply` runs them. A repository already archived is skipped and said so. It stops before step 2 for a repository whose README update failed. It runs from any directory and fails clearly when `gh` is missing or not logged in.

The operator runs it only after a release has published from this repository, which the runbook says; the script does not check that.

It waits for no other task and shares no path with the other two scripts, so it runs in the first layer with them.

## Done when

- `scripts/archive-old-repos.sh` exists and is executable; without `--apply` it changes nothing and prints each step for the three repositories.
- The pointer text names `bytepunx/signet` and the tag scheme.
- `flai check --strict` is clean.

## Notes

Archiving is hard to reverse for the operator's other repositories: the agent never runs `--apply` (`design/conventions/safety.md`).
