---
id: T-0053
type: task
nature: feature
title: Run every component from scripts/test.sh and the Makefile
status: done
parent: S-0007
owner: alex
created: 2026-10-27T22:27:07Z
updated: 2026-10-27T23:55:09Z
transitions:
  - to: ready
    at: 2026-10-27T23:55:07Z
    by: agent-S-0007
  - to: in-progress
    at: 2026-10-27T23:55:07Z
    by: agent-S-0007
  - to: done
    at: 2026-10-27T23:55:09Z
    by: agent-S-0007
stream: S-0007
tags: [ci]
touches: [scripts/test.sh, Makefile, scripts/README.md]
after: [T-0044, T-0046, T-0047, T-0048, T-0049, T-0050, T-0051, T-0052]
usage:
  source: log
  seconds: 2
  estimated: true
  priced_by: harness
  models: []
---
# T-0053 Run every component from scripts/test.sh and the Makefile

## Work

Extend `scripts/test.sh` to run, one line per component, `scripts/test-server.sh`, `test-proto.sh`, `test-client-go.sh`, `test-client-python.sh`, `test-client-typescript.sh`, `test-client-rust.sh`, `test-client-csharp.sh`, and the existing `test-client-elixir.sh`. It runs all of them even when one fails, prints a pass/fail line per component, and exits non-zero if any failed, so one run shows every failure.

Add `test-<component>` targets to the `Makefile` that call each script, kept in the `help` listing, so a developer can run one component with `make test-client-go`; `make test` still calls `scripts/test.sh`.

Update `scripts/README.md`: one row per new script (`buf-generate.sh`, `check-server-chart.sh`, and each `test-*.sh`).

It waits for every script it calls (T-0044, T-0046 to T-0052).

## Done when

- `make test` runs all eight components and exits 0 on a clean checkout.
- With one component's script made to fail, `make test` still runs the others and exits non-zero naming the failed component.
- `make test-client-go` runs only that script.
- `scripts/README.md` lists every script under `scripts/`.

## Notes

Layer 4 of the plan. Docker and network access to the BSR and registries are the only prerequisites; say so in `scripts/README.md`.
