---
id: S-0006
type: story
nature: improvement
title: Baseline component tags and retire release-please
status: done
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:49Z
updated: 2026-10-27T23:01:45Z
transitions:
  - to: ready
    at: 2026-10-27T22:23:53Z
    by: alex
  - to: in-progress
    at: 2026-10-27T22:56:18Z
    by: agent-S-0006
  - to: review
    at: 2026-10-27T22:57:30Z
    by: agent-S-0006
  - to: done
    at: 2026-10-27T23:01:45Z
    by: alex
tags: [server]
topics: [release]
touches: [server/.github/workflows/ci.yml, server/.github/workflows/publish.yml, server/.github/workflows/release-please.yml, server/.github/workflows/release.yml, server/.github/FUNDING.yml, server/release-please-config.json, server/.release-please-manifest.json, proto/.github/workflows/ci.yml, proto/.github/workflows/publish.yml, proto/.github/workflows/release-please.yml, proto/.github/workflows/release.yml, proto/release-please-config.json, proto/.release-please-manifest.json, client-erlang/README.md, system-flow.yaml, client-elixir/README.md, client-elixir/buf.gen.yaml, docs/operators/release.md]
after: [S-0005]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
usage:
  source: log
  seconds: 80
  priced_by: harness
  turns:
    - day: 2026-10-27
      ceremony: 2
      work: 14
  models:
    - model: claude-sonnet-5-5
      input: 38
      output: 6179
      cache_read: 1198210
      cache_write: 87165
      cost: 0.5303
forecast:
  duration: 1h
  delivery: 2026-10-28T06:13:00Z
  basis: "Its own forecast of 1h; 1st in the pull order with an in-progress limit of 2, with nothing ahead of it."
  by: flai
  at: 2026-10-27T22:54:19Z
---
# S-0006 Baseline component tags and retire release-please

## Goal

flai continues each component's version numbering from where the old repositories left off, and nothing of release-please remains: the dead `.github/` folders and release-please files under `server/` and `proto/` are deleted, `client-erlang/` stops being a component (its toolchain note moves into the Elixir client README and the folder and its `system-flow.yaml` entry go), and annotated baseline tags are pushed on the import commits: `server/v0.11.0`, `proto/v0.4.0`, `client-go/v0.7.0`, `client-python/v0.5.0`, `client-rust/v0.6.0`, `client-typescript/v0.9.0`, `client-csharp/v0.6.0`.

## Acceptance criteria
- ~~[ ] `flai release --pending --dry-run` plans no component from `0.0.0` other than `client-elixir`, and plans nothing for `client-erlang`~~
- ~~[ ] The baseline tags exist on the remote on the commits that imported each component, with annotations naming the old repository's tag they continue~~
- [x] No `release-please` file, manifest, or workflow remains anywhere in the repository, and `server/.github/` and `proto/.github/` are gone
- [x] `client-erlang/` is gone, `system-flow.yaml` no longer lists it, and the Elixir client README keeps the note on why Buf's registry cannot generate BEAM stubs

## Tasks
- T-0022 Delete the dead server/.github folder and the server's release-please files
- T-0023 Delete the dead proto/.github folder and the proto module's release-please files
- T-0026 Remove client-erlang as a component and keep its toolchain note in the Elixir client
- T-0033 Write the baseline tag commands for the operator in the release runbook

## Notes

Pushing tags is the operator's action (safety.md); the story prepares the exact commands in the narrative and the runbook. The old root `v*` tags stay as history of the server. Publishing this story releases a patch of `server` and `proto` (deleted files only), which is the first exercise of the new workflows once they exist.

### Planning

Touches, by source:

- Declared (16, all kept): the deleted files under `server/` and `proto/`, `client-erlang/README.md`, `system-flow.yaml`, and `client-elixir/README.md`.
- Co-change: `flai touches suggest` listed only `proto/CHANGELOG.md`, which release-please rewrote on each release. Not added: the story deletes release-please and leaves the changelogs as history.
- Design: `client-elixir/buf.gen.yaml` added, since its comment links to `../client-erlang/README.md`, which the story deletes. `docs/operators/release.md` added, since the story prepares the tag commands in the runbook S-0005 writes.
- Layout: no folder touch is kept. Every touch names a file.

`after: S-0005`, because the runbook file exists on main only when S-0005 is accepted. The other stories of E-0002 that touch `client-elixir/README.md` (S-0016) or `proto/README.md` (S-0010) are not named: they come after this one by the pull order.

Tasks: T-0022 (server), T-0023 (proto), T-0026 (client-erlang), and T-0033 (tag commands) share no path, so they form one layer and run together.

Criteria 1 and 2 depend on the operator pushing the tags. The agent cannot meet them in its worktree and leaves them unticked, with the commands in the runbook.

Forecast: flai gives 1h from the project default (no history of done stories), size 20. Kept: the tasks are small, and the largest, T-0033, reads commit hashes and writes one runbook section.

Cost of delay: not set. The operator's inputs are missing on this story and on E-0002; the plan's thread asks for them.
