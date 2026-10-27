---
id: S-0007
type: story
nature: improvement
title: Root CI workflow that builds and tests every component on pull requests
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:49Z
updated: 2026-10-27T19:48:20Z
transitions: []
tags: []
topics: [ci, github]
touches: [".github/workflows/ci.yml", scripts/test.sh, scripts/test-server.sh, scripts/test-proto.sh, scripts/test-client-go.sh, scripts/test-client-python.sh, scripts/test-client-typescript.sh, scripts/test-client-rust.sh, scripts/test-client-csharp.sh, scripts/README.md, Makefile]
---
# S-0007 Root CI workflow that builds and tests every component on pull requests

## Goal

A root `.github/workflows/ci.yml` runs one job per component on every pull request and push to `main`, each path-filtered to its folder and running the same script a developer runs locally: `scripts/test-<component>.sh`, each running inside Docker as `scripts/test-client-elixir.sh` does, so nothing is installed on a host or a runner beyond Docker. `scripts/test.sh` runs them all for `make test`. The proto job also runs `buf lint` and `buf breaking` against the last `proto/v*` tag; the server job also runs `golangci-lint`, `make build`, the Helm chart lint and render checks from the old `server/.github/workflows/ci.yml`, and the stub-drift check.

## Acceptance criteria
- [ ] `make test` runs every component's behaviour tests and fails if any fails
- [ ] `ci.yml` has a job per component (`server`, `proto`, `client-go`, `client-python`, `client-typescript`, `client-rust`, `client-csharp`, `client-elixir`) that runs only when its folder, its script, or the workflow changed, and a pull request touching one component runs only that job plus the system-flow check
- [ ] Each job's commands are the scripts under `scripts/`, not inline commands
- [ ] The job names are listed in `docs/operators/release.md` as the checks to require on `main`
- [ ] A pull request from this story shows every job green

## Tasks

## Notes

Reference for each component's old CI steps: `server/.github/workflows/ci.yml`, `proto/.github/workflows/ci.yml`, and signet-clients' `ci.yml` on GitHub.
