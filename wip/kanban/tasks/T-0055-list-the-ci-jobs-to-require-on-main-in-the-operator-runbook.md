---
id: T-0055
type: task
nature: improvement
title: List the CI jobs to require on main in the operator runbook
status: backlog
parent: S-0007
owner: alex
created: 2026-10-27T22:27:22Z
updated: 2026-10-27T22:27:22Z
transitions: []
stream: S-0007
tags: [ci]
touches: [docs/operators/release.md]
after: [T-0054]
---
# T-0055 List the CI jobs to require on main in the operator runbook

## Work

In `docs/operators/release.md` (written by story S-0005, so on `main` only once it is accepted), list the `ci.yml` job names (`server`, `proto`, `client-go`, `client-python`, `client-typescript`, `client-rust`, `client-csharp`, `client-elixir`) as the status checks to require on `main`, and the system-flow check's job beside them, in a short table under the branch protection paragraph. Say that a job skipped because its component did not change counts as passing, and that `scripts/test-<component>.sh` is what each runs, so an operator can reproduce a failure locally with `make test-<component>`.

It waits for the ci.yml task (T-0054), so the names it lists are the workflow's.

## Done when

- `docs/operators/release.md` lists all eight job names and the system-flow check as the checks to require on `main`.
- The names match the `name:` of the jobs in `.github/workflows/ci.yml` exactly.
- `flai check --strict` and the markdown lint are clean.

## Notes

Layer 5 of the plan. The story waits on S-0005 for the file to exist on `main`.
