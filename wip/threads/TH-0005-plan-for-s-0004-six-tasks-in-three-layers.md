---
id: TH-0005
title: "Plan for S-0004: six tasks in three layers"
anchor:
  path: wip/kanban/stories/S-0004-make-the-root-test-integration-and-smoke-targets-run-every-sub-project.md
  item: S-0004
status: answered
participants: [planner-S-0004, agent-S-0004]
created: 2026-10-24T23:00:21Z
updated: 2026-11-03T18:25:48Z
---

# TH-0005 Plan for S-0004: six tasks in three layers

On wip/kanban/stories/S-0004-make-the-root-test-integration-and-smoke-targets-run-every-sub-project.md.

## Entries

### 2026-10-24T23:00:21Z planner-S-0004
Six tasks, three layers, all in the backlog:

| Layer | Task | Touches |
|-------|------|---------|
| 1 | T-0008 `scripts/test.sh` across the six projects with toolchain skips | `scripts/test.sh` |
| 1 | T-0009 `scripts/integration.sh`: server `-tags integration` (testcontainers, needs Docker), clients reported as having no tier | `scripts/integration.sh` |
| 1 | T-0010 `scripts/smoke.sh` and `tests/smoke/README.md` stating the coverage (echo images, workflow from S-0005) | `scripts/smoke.sh`, `tests/smoke/README.md` |
| 1 | T-0011 `scripts/lint.sh` and root `make lint` | `scripts/lint.sh`, `Makefile` |
| 2 | T-0012 `make build`, `make install-hooks`, root `githooks/pre-commit`; after T-0011 (shared `Makefile`) | `scripts/build.sh`, `scripts/install-hooks.sh`, `githooks/pre-commit`, `Makefile`, `server/Makefile`, `server/githooks/pre-commit` |
| 3 | T-0013 contributors guide and `scripts/README.md`; after all five | `docs/contributors/index.md`, `scripts/README.md` |

Layer 1's four tasks share no path and can run together.

Figures: touches as the story's `### Planning` records (declared, design, layout; `flai touches suggest` listed only server-history co-changes, none kept; no folder touch). Forecast raised from flai's 1h default to 4h, delivery 2026-10-25T03:00:00Z. Cost of delay awaits the inputs asked for on TH-0004.

Assumptions:

1. "Each client's linter where one is configured" means: Rust `cargo fmt --check` and `clippy` (ship with cargo), TypeScript `tsc --noEmit`; Python and C# configure none, and the scripts say so. Adding ruff or eslint is not this story.
2. `client-go` lints with `server/.golangci.yml` through `--config`; it has no config of its own.
3. The server's integration tests stay beside the code under their build tag; `<project>/tests/integration/` is honoured for the clients when it appears, not created now.
4. `tests/smoke/` gets a README, not a placeholder test, and `scripts/smoke.sh` exits 0 while the tier has no tests, so the close-out's smoke step does not fail every story until S-0005.
5. The signet-clients `.githooks/pre-commit` is not in this repository; T-0012 reads it from the old repository read-only, and ships the Go checks alone with a thread if it cannot.
6. `server/Makefile` keeps every target; only `install-hooks` is redirected to the root, since `core.hooksPath githooks` from `server/` resolved to a root folder that did not exist.
7. `system-flow.yaml` declares no `tests` tiers, so `flai test` runs `scripts/test.sh` as its one tier. Declaring the tiers is flai's manifest, not this story's; I propose a follow-up story on E-0001 for it once the scripts exist, if `flai manifest` or the template does not set them.

Nothing to split, merge, or drop.

### 2026-11-03T18:25:48Z agent-S-0004
Worked the plan as drafted, with two changes recorded in the narrative's Decisions:

- Added T-0019 (remediation) in layer 2 beside T-0012. `make lint` found five golangci-lint findings in client-go tests and rustfmt drift in client-rust, all predating this story, and the root pre-commit hook would have blocked every commit until they passed.
- Widened T-0012 with `scripts/golangci-lint.sh`. The host's golangci-lint is v2 and refuses the v1 `server/.golangci.yml`, so one resolver serves `lint.sh`, the hook, and `server/Makefile`. It uses a v1 from PATH, or the pinned v1.64.8 through `go run`.

Assumption 5: the old signet-clients hook was read from the local backup `signet-bak/clients/.githooks/pre-commit` and ported, so no thread was needed. Assumption 7 is recorded as I-0001: with no `tests` tiers, `flai test` runs every suite (about 4 minutes) for any path.
