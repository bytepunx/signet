---
id: TH-0005
title: Plan for S-0020 and the cost of delay inputs it needs
anchor:
  path: wip/kanban/stories/S-0020-generate-in-repo-stubs-from-proto-by-directory-instead-of-the-bsr-pin.md
  item: S-0020
status: resolved
participants: [planner-S-0020, alex]
created: 2026-10-27T22:26:44Z
updated: 2026-11-06T08:31:56Z
---

# TH-0005 Plan for S-0020 and the cost of delay inputs it needs

On wip/kanban/stories/S-0020-generate-in-repo-stubs-from-proto-by-directory-instead-of-the-bsr-pin.md.

## Entries

### 2026-10-27T22:26:44Z planner-S-0020
Question first. S-0020 and E-0002 have no cost of delay inputs, so `flai cod` cannot give a value. I recommend `time_lost_per_cycle` of 30m and no revenue or penalty: until this lands, every schema change needs a manual BSR-pin bump in up to seven components, and the old repository's proto-bump automation is gone. Alternatives: a `penalty_per_week` figure if you put a price on it, or no value, which leaves the story at the bottom of cost-of-delay ordering. Please set the inputs with `flai edit S-0020 --time-lost-per-cycle 30m` (or the figure you choose); I will work out the value on my next run.

Plan: eight tasks in two layers.

- Layer 1, all independent (no shared paths): T-0028 server, T-0029 Go client, T-0031 Python client (with its echo Dockerfile), T-0032 TypeScript client, T-0034 Rust client (with its echo Dockerfile), T-0035 C# client (with its echo Dockerfile), T-0036 Elixir client (with `scripts/test-client-elixir.sh`).
- Layer 2: T-0038 rewrites `design/tech/buf.md` and checks the whole story (no `module:` input left, `make test` passes). It waits for all of layer 1.

Story `after`: S-0005 (the ADR deciding directory input), S-0007 (the root CI and test scripts behind `make test` for every component), S-0011 (changes the Go client's module path and `go_package_prefix` in `client-go/buf.gen.yaml`, which the regenerated stubs must match).

Assumptions:

- The Python, Rust, and C# echo Dockerfiles copy only `buf.gen.yaml` before `buf generate`, so a relative directory input breaks them. I assumed they should build from the repository root, copying `proto/proto` and the component's `buf.gen.yaml`. S-0019 (echo image smoke builds) will need to use that context; it has no workflow yet.
- The Elixir test script mounts only `client-elixir/`, so it needs the repository root mounted.
- `server/CLAUDE.md` (S-0017), the server CI workflow (S-0006, S-0007), and `proto/README.md` (S-0010) are other stories' files; I left them out of the touches. `server/CLAUDE.md` will still mention the BSR until S-0017 lands.
- "Identical stubs" is checked as an empty `git diff` of the committed stubs (server, Go, TypeScript) after `buf generate`, and by comparing outputs for the clients that do not commit them. It needs network access to the BSR's remote plugins.
- Forecast 3h, up from the 1h default, because of the seven generate-and-diff runs and the three Dockerfile rewrites.

No story to split, merge, add, or drop.

### 2026-11-06T08:31:56Z alex
Resolved.
