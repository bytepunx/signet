---
id: T-0024
type: task
nature: improvement
title: "Script: set and check the repository's release secrets"
status: done
parent: S-0008
owner: alex
created: 2026-10-27T22:25:18Z
updated: 2026-10-28T05:52:27Z
transitions:
  - to: ready
    at: 2026-10-28T05:52:05Z
    by: agent-S-0008
  - to: in-progress
    at: 2026-10-28T05:52:05Z
    by: agent-S-0008
  - to: done
    at: 2026-10-28T05:52:27Z
    by: agent-S-0008
stream: S-0008
tags: [release, github]
touches: [scripts/set-release-secrets.sh]
usage:
  source: log
  seconds: 22
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 1
      output: 267
      cache_read: 39330
      cache_write: 1917
      cost: 0.0143
---
# T-0024 Script: set and check the repository's release secrets

## Work

Write `scripts/set-release-secrets.sh`, POSIX `sh` under `set -eu`, quoted for zsh and bash. It manages the Actions secrets of `bytepunx/signet` that `design/system/release.md` names: `BUF_TOKEN`, `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`, `CARGO_REGISTRY_TOKEN`, `NUGET_USER`, and `HEX_API_KEY`.

- `check` (the default) runs `gh secret list --repo bytepunx/signet` and prints each name as present or missing; it exits non-zero while any is missing.
- `set` reads each value from an environment variable of the same name, never from an argument, and passes it to `gh secret set NAME --repo bytepunx/signet --body -` on standard input. It skips a name whose variable is empty and says so. It never prints a value.
- The repository is `bytepunx/signet` unless `SIGNET_REPO` overrides it.
- It prints what it does, runs from any directory, and fails with a clear message when `gh` is missing or not logged in.

It waits for no other task and shares no path with the other two scripts, so it runs in the first layer with them.

## Done when

- `scripts/set-release-secrets.sh` exists, is executable, and passes `shellcheck` when it is installed.
- Run with no `gh` on the path, it exits non-zero and says why.
- Its output never contains a secret value, which a run with a dummy value in the environment shows.
- `flai check --strict` is clean.

## Notes

The agent never runs `set` with real values: creating credentials is the operator's (`design/conventions/safety.md`).
