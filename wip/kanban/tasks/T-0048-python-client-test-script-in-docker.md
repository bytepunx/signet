---
id: T-0048
type: task
nature: feature
title: Python client test script in Docker
status: backlog
parent: S-0007
owner: alex
created: 2026-10-27T22:26:24Z
updated: 2026-10-27T22:26:24Z
transitions: []
stream: S-0007
tags: [client-python, ci]
touches: [scripts/test-client-python.sh]
after: [T-0042]
---
# T-0048 Python client test script in Docker

## Work

Add `scripts/test-client-python.sh`, POSIX `sh` under `set -eu`, modelled on `scripts/test-client-elixir.sh`: run `scripts/buf-generate.sh client-python` (stubs are gitignored under `src/signet/` and `src/admin/`), then with `client-python/` mounted in a pinned `python` image of the lowest Python `pyproject.toml` supports (3.10 or newer; pick one and record it) create a virtualenv outside the mount, `pip install -e '.[dev,workload]'`, and run `pytest`. Extra arguments run instead of `pytest`.

It waits for `scripts/buf-generate.sh` (T-0042).

## Done when

- `scripts/test-client-python.sh` exits 0 on a clean checkout and runs everything under `client-python/tests/`.
- A failing test makes it exit non-zero.
- Nothing but Docker is installed on the host, no virtualenv or `.egg-info` is left in the working tree beyond gitignored paths, and `sh -n` is clean.

## Notes

Layer 2 of the plan.
