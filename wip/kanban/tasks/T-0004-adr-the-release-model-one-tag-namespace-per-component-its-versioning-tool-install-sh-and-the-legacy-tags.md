---
id: T-0004
type: task
nature: research
title: "ADR: the release model, one tag namespace per component, its versioning tool, install.sh, and the legacy tags"
status: done
parent: S-0001
owner: alex
created: 2026-10-24T22:52:55Z
updated: 2026-11-03T18:25:12Z
transitions:
  - to: ready
    at: 2026-11-03T18:21:33Z
    by: agent-S-0001
  - to: in-progress
    at: 2026-11-03T18:21:33Z
    by: agent-S-0001
  - to: done
    at: 2026-11-03T18:25:12Z
    by: agent-S-0001
stream: S-0001
tags: [adr, release]
touches: [design/adrs/README.md, design/adrs/0003-each-component-releases-under-its-own-project-vx-y-z-tag-computed-by-flai.md]
after: [T-0003]
usage:
  source: log
  seconds: 219
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 46
      output: 15067
      cache_read: 2793181
      cache_write: 53696
      cost: 1.2233
---
# T-0004 ADR: the release model, one tag namespace per component, its versioning tool, install.sh, and the legacy tags

## Work

Read `design/conventions/git.md` (publishing is `flai release --pending`, per-component bumps by story nature), `server/release-please-config.json`, `proto/release-please-config.json`, `server/.github/workflows/release-please.yml` and `release.yml`, `proto/.github/workflows/release-please.yml` and `release.yml`, `install.sh` (which resolves `releases/latest` of `bytepunx/signet`), the old signet-clients `release-please-config.json` and `publish-*.yml` on GitHub, and the tag list (`git tag`): `v*` from the old server, `legacy/<project>/*`, and one `<project>/v*` per project. Decide: the tag namespace per component (`server/v*`, `proto/v*`, `client-<lang>/v*`, matching the Go module directories T-0003 decided); whether `flai release --pending` or release-please with one multi-component manifest computes versions and writes `CHANGELOG.md`, confirming the convention or recording the conflict with it; how `install.sh` finds the latest `server/v*` release; what happens to the `v*` and `legacy/*` tags (kept as history, or deleted). Record it with `flai adr new --commit` from the story's worktree. Say in Consequences what S-0005 and S-0006 must build.

## Done when

- One ADR under `design/adrs/`, status `accepted`, decides the tag namespaces, the versioning and changelog tool, the `install.sh` lookup, and the legacy tags, with the losing alternatives each given one line.
- The ADR says whether `design/conventions/git.md` stands or needs a project addition, without editing the convention's baseline.
- `design/adrs/README.md` lists it and `flai check --strict` is clean.

## Notes

Waits for T-0003: the tag namespaces follow the Go module directories, and the ADR tasks share `design/adrs/README.md`.
