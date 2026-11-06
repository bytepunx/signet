---
id: I-0012
title: Concurrent flai test runs in one worktree corrupt client-elixir/deps, so later runs fail in mix compile
class: efficiency
status: open
count: 1
cost: 10m
first_reported: 2026-11-06T08:34:44Z
last_reported: 2026-11-06T08:34:44Z
updated: 2026-11-06T08:34:44Z
---

# I-0012 Concurrent flai test runs in one worktree corrupt client-elixir/deps, so later runs fail in mix compile

## Description
Concurrent flai test runs in one worktree corrupt client-elixir/deps, so later runs fail in mix compile

## Instances

### 2026-11-06T08:34:44Z
Story: S-0011.
S-0011, 2026-11-06T08:33Z: three task sub-agents ran flai test in .flai-cache/worktrees/S-0011 within two minutes; each path under a component selects scripts/test.sh, the whole tier, so three scripts/test-client-elixir.sh runs did mix deps.get and mix compile at once on the same host-mounted client-elixir/deps and _build. One run's fetch raced another's, deps/grpc was left empty, and the close-out then stopped at test with 'could not compile dependency :grpc' and 'Could not find a Mix.Project'. Fixed by removing client-elixir/deps and _build in the worktree and asking for the close-out again. Remediation: keep each Elixir run's deps and _build in a named Docker volume, as the Go and Rust scripts do for their caches, or have flai test select a per-component tier so a change under client-go does not run the Elixir tier.

## Remediation
