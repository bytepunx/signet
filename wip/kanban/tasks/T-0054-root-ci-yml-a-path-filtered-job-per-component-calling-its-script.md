---
id: T-0054
type: task
nature: feature
title: "Root ci.yml: a path-filtered job per component calling its script"
status: backlog
parent: S-0007
owner: alex
created: 2026-10-27T22:27:18Z
updated: 2026-10-27T22:27:18Z
transitions: []
stream: S-0007
tags: [ci]
touches: [".github/workflows/ci.yml"]
after: [T-0044, T-0046, T-0047, T-0048, T-0049, T-0050, T-0051, T-0052]
---
# T-0054 Root ci.yml: a path-filtered job per component calling its script

## Work

Add `.github/workflows/ci.yml`, named `CI`, on `pull_request` to `main` and `push` to `main`, with `permissions: contents: read`, pinned action versions, and a `concurrency` group that cancels superseded pull request runs.

- A `changes` job uses a path filter action (`dorny/paths-filter`, pinned) to report which components changed. Each component's filter is its folder, its `scripts/test-<component>.sh`, and `.github/workflows/ci.yml`; the generating components add `scripts/buf-generate.sh`, and `server` adds `scripts/check-server-chart.sh`. Do not add `proto/**` to the others: story S-0020 does that when the clients generate from `proto/`.
- One job per component, named exactly `server`, `proto`, `client-go`, `client-python`, `client-typescript`, `client-rust`, `client-csharp`, `client-elixir`, each `needs: changes` and `if:` its filter output. A job skipped by `if:` reports success, so the names can be required checks on `main` without blocking pull requests that leave the component alone (a workflow-level `paths:` filter would leave the check pending).
- Each job checks out the repository (`proto` with `fetch-depth: 0` for its tags), and runs one command: `scripts/test-<component>.sh`. Docker is on `ubuntu-latest`; no language toolchain is set up.
- Pass `BUF_TOKEN` from the secrets to the jobs that generate, when it is set.
- The system-flow check stays in `.github/workflows/system-flow-check.yml`; do not duplicate it.

It waits for the scripts it calls (T-0044, T-0046 to T-0052) so that every job it names exists and runs green.

## Done when

- `ci.yml` passes `actionlint` (run it with Docker, `rhysd/actionlint`) and `flai check --strict`.
- It has the eight jobs above and each runs only `scripts/test-<component>.sh`.
- A change to only `client-go/` selects only the `client-go` job in the filter's output, and a change only under `docs/` selects none.
- The workflow's pull request run on this story's branch shows every job green, or skipped for a component the branch does not change.

## Notes

Layer 4 of the plan.
