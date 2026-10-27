---
id: T-0060
type: task
nature: feature
title: Script that writes the tag's version into pyproject.toml
status: backlog
parent: S-0012
owner: alex
created: 2026-10-27T22:52:10Z
updated: 2026-10-27T22:52:10Z
transitions: []
stream: S-0012
tags: [client-python]
touches: [scripts/set-client-python-version.sh]
---
# T-0060 Script that writes the tag's version into pyproject.toml

## Work

Add `scripts/set-client-python-version.sh`, POSIX `sh` under `set -eu`, so the version step of the release workflow is one named, locally testable command, as `design/conventions/tooling.md` asks.

- Usage: `scripts/set-client-python-version.sh <tag-or-version>`. It accepts `client-python/v0.6.0`, `v0.6.0`, or `0.6.0`, strips the `client-python/` and `v` prefixes, and prints the version it wrote.
- Refuse a version that is not `X.Y.Z` with an optional PEP 440 pre-release suffix (`a1`, `b2`, `rc1`), and exit non-zero with a message that names the input. A tag such as `client-python/v0.6.0-test` is refused, because PyPI would reject it after the build.
- Rewrite the top-level `version = "..."` line of `client-python/pyproject.toml` in place, found from the script's own location (`cd "$(dirname "$0")/.."`), working with both GNU and BSD `sed`. Fail if the file has no such line or more than one.
- Print nothing else to standard output, so a workflow can capture the version with `$(...)`; diagnostics go to standard error.

It waits for no task: it only needs the `version` line to exist, which it does today.

## Done when

- Run on a scratch copy of `pyproject.toml`: `client-python/v0.6.0`, `v0.6.0`, and `0.6.0` each leave `version = "0.6.0"`, and `0.6.0rc1` leaves `version = "0.6.0rc1"`.
- `client-python/v0.6.0-test`, an empty argument, and no argument each exit non-zero and leave the file unchanged.
- The script is executable and `flai check --strict` is clean.

## Notes
