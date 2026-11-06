---
id: S-0004
type: story
nature: improvement
title: Make the root test, integration, and smoke targets run every sub-project
status: review
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-11-03T18:31:41Z
transitions:
  - to: ready
    at: 2026-10-24T22:33:47Z
    by: claude
  - to: in-progress
    at: 2026-11-03T18:12:15Z
    by: agent-S-0004
  - to: review
    at: 2026-11-03T18:31:41Z
    by: agent-S-0004
tags: [server, tooling, make, scripts, tests]
topics: [testing, tooling]
touches: [Makefile, scripts/test.sh, scripts/integration.sh, scripts/smoke.sh, scripts/lint.sh, scripts/build.sh, scripts/install-hooks.sh, scripts/README.md, tests/smoke/README.md, githooks/pre-commit, server/Makefile, server/githooks/pre-commit, docs/contributors/index.md, client-go/client_test.go, client-go/restart_test.go, client-rust/examples/echo.rs, client-rust/examples/getsecret.rs, client-rust/examples/restart_on_change.rs, client-rust/src/client.rs, client-rust/src/lib.rs, client-rust/src/restart.rs, scripts/golangci-lint.sh, design/issues/I-0001-flai-test-runs-every-sub-project-s-behaviour-tests-for-any-path-because-the.md, design/issues/summary.md, design/issues/I-0002-flai-check-finds-doc-topics-outside-the-story-at-close-out.md, design/issues/I-0003-flai-check-finds-roles-baseline-outside-the-story-at-close-out.md, design/issues/I-0005-buf-generate-against-the-buf-schema-registry-fails-the-test-tier-when-the-bsr.md]
usage:
  source: log
  seconds: 1179
  priced_by: harness
  turns:
    - day: 2026-11-03
      ceremony: 2
      test_runs: 6
      hand_edits: 4
      work: 65
  models:
    - model: claude-opus-5-5
      input: 300
      output: 77217
      cache_read: 19350713
      cache_write: 277997
      cost: 7.405
  strategic:
    - kind: orchestrator
      seconds: 48
      estimated: true
      models:
        - model: claude-opus-5-5
          input: 24
          output: 168
          cache_read: 684127
          cache_write: 108192
          cost: 0.446
forecast:
  duration: 4h
  delivery: 2026-10-25T03:00:00Z
  basis: "Six tasks in three layers across six toolchains, each root script verified on this host where its toolchain is present; flai's 1h default raised to 4h for the six scripts, the root githooks recovery, and the documentation."
  by: planner-S-0004
  at: 2026-10-24T22:58:55Z
---
# S-0004 Make the root test, integration, and smoke targets run every sub-project

## Goal

`make test`, `make integration`, and `make smoke` at the repository root run the real test tiers of every sub-project that has code, through purpose-named scripts in `scripts/`, so the conventions' three tiers are true for this repository and CI can call one entry point per tier.

## Acceptance criteria
- [x] `scripts/test.sh` runs the behaviour tests of `server` (`go test`), `client-go`, `client-python` (pytest), `client-rust` (`cargo test`), `client-typescript` (its package test script), and `client-csharp` (`dotnet test`), fails if any fails, and skips a project only when its toolchain is absent, saying so.
- [x] `scripts/integration.sh` runs `server`'s `-tags integration` tests and any client integration tests, and says plainly when a tier has no tests.
- [x] `scripts/smoke.sh` has a real body or states what smoke coverage exists and where it will come from; `tests/smoke/` exists at the root per `design/conventions/code-quality.md`.
- [x] `make lint` at the root runs `golangci-lint` for the Go projects, `buf lint` for `proto/`, and each client's linter where one is configured.
- [x] `make build` at the root builds `signetd` and `signet` into `server/bin`, and `make install-hooks` installs the pre-commit hooks that `server/githooks` and the old signet-clients `.githooks` provided, from one root `githooks/`.
- [x] `docs/contributors/index.md` lists the root targets and what each runs.

## Tasks
- T-0008 scripts/test.sh runs the behaviour tests of every sub-project with code
- T-0009 scripts/integration.sh runs the server's -tags integration tests and reports the clients' tier
- T-0010 scripts/smoke.sh states the smoke coverage and tests/smoke/ exists at the root
- T-0011 scripts/lint.sh and a root make lint target run golangci-lint, buf lint, and the configured client linters
- T-0012 Root make build and make install-hooks, with one githooks/pre-commit for the Go projects
- T-0013 Document the root targets in docs/contributors/index.md and scripts/README.md
- T-0019 Fix the existing client-go lint findings and client-rust formatting so make lint passes
- T-0020 scripts/test.sh regenerates client stubs only when they are missing or older than buf.gen.yaml

## Notes

`server/Makefile` already has `build`, `test`, `test-int`, `lint`, `proto`, `install-hooks`; keep it as the per-project entry point and call it from the root scripts rather than duplicating it. `client-erlang` and `admin` have no code and are skipped.

### Planning

Planned by planner-S-0004 on 2026-10-24; see TH-0004 and the plan thread on this story.

Touches, by origin:

| Touch | Origin |
|-------|--------|
| `scripts/test.sh`, `scripts/integration.sh`, `scripts/smoke.sh`, `docs/contributors/index.md` | declared: named in the criteria |
| `Makefile` | design: criteria 4 and 5 add the root `lint`, `build`, and `install-hooks` targets; `tooling.md` says Make targets call scripts |
| `scripts/lint.sh`, `scripts/build.sh`, `scripts/install-hooks.sh` | design: `tooling.md` and `scripts/README.md` want one purpose-named script per target |
| `scripts/README.md` | layout: its table lists every script and must gain the three new ones |
| `tests/smoke/README.md` | design: criterion 3 and `code-quality.md` put smoke tests in `tests/smoke/` at the root; a README is the file that makes the folder exist and says what it is for |
| `githooks/pre-commit` | design: criterion 5 asks for one root `githooks/` |
| `server/Makefile`, `server/githooks/pre-commit` | layout: `server/Makefile`'s `install-hooks` sets `core.hooksPath githooks`, which git resolves against the repository root, where no `githooks/` exists; the root hook replaces the server one and the server target points at it |

`flai touches suggest` seeded with the declared paths listed only files of the imported server history (`.github/workflows/ci.yml`, `gen/**`, `internal/**`): co-changes of the server's own build files before the import, not of the root scripts. None kept. No folder touch: every file the story adds is named by a task.

Forecast: `flai forecast` gave 1h, `planning.default_duration` for size 6 with no history. Raised to 4h: six scripts across six toolchains, each verified by running it here (dotnet is absent on this host, so the skip path is what gets verified for C#), recovery of the signet-clients `.githooks` that the import dropped, and two documents. Delivery 2026-10-25T03:00:00Z follows from that duration from now, second in the pull order behind S-0001 with an in-progress limit of 2.

Cost of delay: no inputs on S-0004 or E-0001; asked on TH-0004 (and TH-0001 asks the same for E-0001). The value is written once the operator sets the inputs.
