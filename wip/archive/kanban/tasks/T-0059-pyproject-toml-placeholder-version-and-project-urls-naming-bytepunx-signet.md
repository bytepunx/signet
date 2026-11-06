---
id: T-0059
type: task
nature: improvement
title: "pyproject.toml: placeholder version and project URLs naming bytepunx/signet"
status: done
parent: S-0012
owner: alex
created: 2026-10-27T22:52:07Z
updated: 2026-11-06T07:51:28Z
transitions:
  - to: ready
    at: 2026-11-06T07:51:26Z
    by: agent-S-0012
  - to: in-progress
    at: 2026-11-06T07:51:26Z
    by: agent-S-0012
  - to: done
    at: 2026-11-06T07:51:28Z
    by: agent-S-0012
stream: S-0012
tags: [client-python]
touches: [client-python/pyproject.toml]
usage:
  source: log
  seconds: 2
  estimated: true
  priced_by: harness
  models: []
---
# T-0059 pyproject.toml: placeholder version and project URLs naming bytepunx/signet

## Work

Make `client-python/pyproject.toml` fit the release model of `design/system/release.md`: the checked-in `version` is a placeholder the release workflow overrides, and the metadata names this repository.

- Set `version = "0.0.0"` on its own line, with a comment above it saying the release workflow rewrites it from the `client-python/vX.Y.Z` tag with `scripts/set-client-python-version.sh` and that no commit bumps it. Keep the `version = "..."` line at the top level of `[project]`, since the script matches it there.
- Add a `[project.urls]` table: `Homepage` and `Repository` as `https://github.com/bytepunx/signet` and `https://github.com/bytepunx/signet/tree/main/client-python`, `Issues` as `https://github.com/bytepunx/signet/issues`, and `Releases` as `https://github.com/bytepunx/signet/releases`.
- Change the `description` so it no longer names `bytepunx/signet-proto` as the source repository: "Python client for signet, generated from the signet proto schema".
- Leave the dependencies, the build backend, and the wheel `packages` list as they are. hatch-vcs is not used: its tag lookup does not fit a `client-python/v` prefix in a repository of many components, and a one-line rewrite is simpler to check.

No other file changes. This task has no `after`: it runs with the script task, and the workflow task waits for both.

## Done when

- `pyproject.toml` has exactly one top-level `version = "0.0.0"` line under `[project]` with the comment above it.
- `[project.urls]` names `bytepunx/signet` and no URL names `signet-clients` or `signet-proto`.
- `python -c "import tomllib; tomllib.load(open('client-python/pyproject.toml','rb'))"` parses the file.
- `flai check --strict` is clean.

## Notes
