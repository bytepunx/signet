---
id: S-0007
type: story
nature: improvement
title: Root CI workflow that builds and tests every component on pull requests
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:49Z
updated: 2026-10-27T22:48:44Z
transitions: []
tags: []
topics: [ci, github]
touches: [".github/workflows/ci.yml", scripts/test.sh, scripts/test-server.sh, scripts/test-proto.sh, scripts/test-client-go.sh, scripts/test-client-python.sh, scripts/test-client-typescript.sh, scripts/test-client-rust.sh, scripts/test-client-csharp.sh, scripts/README.md, Makefile, scripts/buf-generate.sh, scripts/check-server-chart.sh, docs/operators/release.md]
after: [S-0005]
forecast:
  duration: 6h
  delivery: 2026-10-28T06:34:00Z
  basis: "Its own forecast of 6h; 2nd in the pull order with an in-progress limit of 2, behind S-0006."
  by: flai
  at: 2026-10-27T22:48:44Z
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
- T-0042 Shared buf generate script that runs in Docker
- T-0044 Proto test script: buf lint and buf breaking against the last proto tag
- T-0046 Server test script: go test, golangci-lint, make build, and stub drift in Docker
- T-0047 Go client test script in Docker
- T-0048 Python client test script in Docker
- T-0049 TypeScript client test script in Docker
- T-0050 Rust client test script in Docker
- T-0051 C# client test script in Docker
- T-0052 Server Helm chart lint and render checks script
- T-0053 Run every component from scripts/test.sh and the Makefile
- T-0054 Root ci.yml: a path-filtered job per component calling its script
- T-0055 List the CI jobs to require on main in the operator runbook

## Notes

Reference for each component's old CI steps: `server/.github/workflows/ci.yml`, `proto/.github/workflows/ci.yml`, and signet-clients' `ci.yml` on GitHub.

### Planning

Touches. The eleven declared touches are kept. Three are added, each a file: `scripts/buf-generate.sh` (design: the clients and the server generate stubs from `buf.gen.yaml`, and one Docker script avoids seven copies), `scripts/check-server-chart.sh` (design: the goal names the Helm checks, kept apart so `scripts/test-server.sh` stays readable), and `docs/operators/release.md` (design: criterion 4 names it). `flai touches suggest` listed only co-change files of the server's history (generated stubs, Helm chart, gitops code); none belongs to this story, so none was taken. No folder touch is kept.

After. `S-0005` only: `docs/operators/release.md` and `design/system/release.md` exist on main only once S-0005 is accepted, and criterion 4 edits the former. S-0006 (baseline tags) is not needed on main: `scripts/test-proto.sh` skips `buf breaking` when no `proto/v*` tag exists.

Layers of tasks:

| Layer | Tasks |
|-------|-------|
| 1 | T-0042 buf-generate.sh, T-0044 test-proto.sh |
| 2 | T-0046 test-server.sh, T-0047 Go, T-0048 Python, T-0049 TypeScript, T-0050 Rust, T-0051 C# |
| 3 | T-0052 check-server-chart.sh and its call from test-server.sh |
| 4 | T-0053 test.sh, Makefile, scripts/README.md; T-0054 ci.yml |
| 5 | T-0055 docs/operators/release.md |

Forecast. `flai forecast` gave the default 1h for size 16 (5 criteria, 11 touches) with no history to draw on. Adjusted to 6h: eight Docker scripts that each need a pinned image, a working `buf generate` against the BSR, and a local run (about 30 to 40 minutes each), plus the workflow and the runbook edit. Delivery is the start after S-0006 (about 23:25Z) plus 6h.

Cost of delay. The operator's inputs are missing; they are asked for on the plan thread, and the value is written once they exist.
