---
id: S-0004
type: story
nature: improvement
title: Make the root test, integration, and smoke targets run every sub-project
status: ready
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-10-24T22:33:47Z
transitions:
  - to: ready
    at: 2026-10-24T22:33:47Z
    by: claude
tags: []
---
# S-0004 Make the root test, integration, and smoke targets run every sub-project

## Goal

`make test`, `make integration`, and `make smoke` at the repository root run the real test tiers of every sub-project that has code, through purpose-named scripts in `scripts/`, so the conventions' three tiers are true for this repository and CI can call one entry point per tier.

## Acceptance criteria
- [ ] `scripts/test.sh` runs the behaviour tests of `server` (`go test`), `client-go`, `client-python` (pytest), `client-rust` (`cargo test`), `client-typescript` (its package test script), and `client-csharp` (`dotnet test`), fails if any fails, and skips a project only when its toolchain is absent, saying so.
- [ ] `scripts/integration.sh` runs `server`'s `-tags integration` tests and any client integration tests, and says plainly when a tier has no tests.
- [ ] `scripts/smoke.sh` has a real body or states what smoke coverage exists and where it will come from; `tests/smoke/` exists at the root per `design/conventions/code-quality.md`.
- [ ] `make lint` at the root runs `golangci-lint` for the Go projects, `buf lint` for `proto/`, and each client's linter where one is configured.
- [ ] `make build` at the root builds `signetd` and `signet` into `server/bin`, and `make install-hooks` installs the pre-commit hooks that `server/githooks` and the old signet-clients `.githooks` provided, from one root `githooks/`.
- [ ] `docs/contributors/index.md` lists the root targets and what each runs.

## Tasks

## Notes

`server/Makefile` already has `build`, `test`, `test-int`, `lint`, `proto`, `install-hooks`; keep it as the per-project entry point and call it from the root scripts rather than duplicating it. `client-erlang` and `admin` have no code and are skipped.
