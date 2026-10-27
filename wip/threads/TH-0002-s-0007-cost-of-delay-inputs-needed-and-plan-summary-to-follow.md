---
id: TH-0002
title: "S-0007: cost of delay inputs needed (and plan summary to follow)"
anchor:
  path: wip/kanban/stories/S-0007-root-ci-workflow-that-builds-and-tests-every-component-on-pull-requests.md
  item: S-0007
status: open
participants: [planner-S-0007]
created: 2026-10-27T22:24:37Z
updated: 2026-10-27T22:27:59Z
---

# TH-0002 S-0007: cost of delay inputs needed (and plan summary to follow)

On wip/kanban/stories/S-0007-root-ci-workflow-that-builds-and-tests-every-component-on-pull-requests.md.

## Entries

### 2026-10-27T22:24:37Z planner-S-0007
Recommended answer first: set no revenue, a modest penalty, and a time-lost input on S-0007 (or on E-0002 for all its stories): revenue_per_week 0, penalty_per_week 0, time_lost_per_cycle 4h. Reason: S-0007 has no revenue; its cost of delay is that S-0009 to S-0016 (every release workflow) wait on it and nothing is tested on pull requests until it lands, which costs about one manual test round per cycle.

`flai cod S-0007` refuses: neither S-0007 nor E-0002 has cost of delay inputs, and the inputs are yours. Set them with `flai edit S-0007 --revenue-per-week N --penalty-per-week N --time-lost-per-cycle 4h`, or answer here with the figures. I plan the tasks, touches and forecast meanwhile and write the cost of delay value once the inputs exist.

### 2026-10-27T22:27:59Z planner-S-0007
Plan for S-0007: 12 tasks in 5 layers. The cost of delay inputs above are still needed.

| Layer | Tasks |
|-------|-------|
| 1 | T-0042 `scripts/buf-generate.sh` (buf 1.71.0 in Docker, repo root mounted); T-0044 `scripts/test-proto.sh` (lint, breaking against the last `proto/v*` tag, skipped when none) |
| 2 | T-0046 server (go test -race, golangci-lint, make build, stub drift); T-0047 Go; T-0048 Python; T-0049 TypeScript; T-0050 Rust; T-0051 C#. Each waits for T-0042 |
| 3 | T-0052 `scripts/check-server-chart.sh` (the Helm checks), called from test-server.sh; waits for T-0046 |
| 4 | T-0053 `scripts/test.sh`, Makefile `test-<component>` targets, `scripts/README.md`; T-0054 `.github/workflows/ci.yml`. Both wait for every script |
| 5 | T-0055 job names in `docs/operators/release.md`; waits for T-0054 |

Story changes: added `after: S-0005` (release.md exists on main only once S-0005 is accepted; `flai check` warns about the `after` hub on S-0005 and S-0007, which is real here), touches grew by `scripts/buf-generate.sh`, `scripts/check-server-chart.sh`, `docs/operators/release.md`. Forecast 6h (default was 1h with no history), delivery 2026-10-28T05:30Z.

Assumptions:
- ci.yml uses a `changes` job with `dorny/paths-filter` and an `if:` on each component job. A job skipped by `if:` counts as passing, so the eight job names can be required on `main`. A workflow-level `paths:` filter would leave required checks pending.
- Clients and the server still generate from the BSR pin (`v0.4.0`), so each job's filter omits `proto/**`. S-0020 (directory input) must add `proto/**` to those filters when it lands; it will need ci.yml in its touches.
- `server/.golangci.yml` is the v1 schema and the pre-commit hook pins v1.64.8, so T-0046 pins the v1 image, not `latest` as the old job did.
- The old `docker-build` job (build signetd and signet images, no push) and the auto-commit of regenerated stubs are not carried over: S-0009 builds the images on release, and a drift check that fails is safer than a bot commit. Say if you want either kept.
- The `docs/` sentence "A pull request from this story shows every job green" (criterion 5) can only be checked on GitHub after the branch is pushed.
- Python/Rust/C#/TypeScript tests need network for the BSR and package registries; Docker is the only host tool.

I propose no split, merge, or drop.
