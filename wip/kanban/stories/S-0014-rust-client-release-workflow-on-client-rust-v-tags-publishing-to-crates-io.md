---
id: S-0014
type: story
nature: feature
title: "Rust client: release workflow on client-rust/v* tags publishing to crates.io"
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-10-27T23:01:46Z
transitions: []
tags: [client-rust]
topics: [release, github]
touches: [".github/workflows/release-client-rust.yml", client-rust/Cargo.toml, client-rust/README.md]
after: [S-0007]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
forecast:
  duration: 2h
  delivery: 2026-10-29T12:56:00Z
  basis: "Its own forecast of 2h; 8th in the pull order with an in-progress limit of 2, behind S-0007, S-0008, S-0009, S-0010, S-0020, S-0011, S-0012 and S-0013."
  by: flai
  at: 2026-10-27T23:01:46Z
---
# S-0014 Rust client: release workflow on client-rust/v* tags publishing to crates.io

## Goal

A push of `client-rust/vX.Y.Z` sets the crate version from the tag, builds, tests, publishes `signet-client` to crates.io, and creates the GitHub Release. `Cargo.toml` names this repository.

## Acceptance criteria
- [ ] `release-client-rust.yml` triggers on `client-rust/v*` only, sets the version in `Cargo.toml` from the tag before `cargo publish`, publishes with `CARGO_REGISTRY_TOKEN` or crates.io trusted publishing, and creates the GitHub Release with notes from the tag annotation
- [ ] `Cargo.toml` repository and documentation fields name `bytepunx/signet`
- [ ] `client-rust/README.md` names the tag form and the workflow

## Tasks
- T-0061 Write release-client-rust.yml: version from the tag, test, publish to crates.io, GitHub Release
- T-0062 Cargo.toml: name bytepunx/signet in repository, documentation, and description
- T-0063 client-rust README: name the client-rust/v tag form and the release workflow
- T-0065 Run the workflow on a test tag client-rust/v0.0.0-test and delete it

## Notes

Reference: signet-clients' `publish-rust.yml` on GitHub.

### Planning

Touches, by source:

- Declared: `.github/workflows/release-client-rust.yml`, `client-rust/Cargo.toml`, `client-rust/README.md`. All kept, all files.
- Co-change: `flai touches suggest` listed `client-rust/CHANGELOG.md` (release-please rewrote it on each release) and `client-rust/src/client.rs` and `src/lib.rs` (feature work). None added: the changelog stays as history and no source changes here.
- Design: `design/system/release.md` sets the workflow's shape (version from the tag, GitHub Release from the annotation and commits). `scripts/test-client-rust.sh` is created by S-0007 and only called here.
- Layout: no folder touch is kept.

After: `S-0007` only, because T-0061 runs `scripts/test-client-rust.sh`, which that story creates. S-0020 (which changes `client-rust/buf.gen.yaml` to a directory input) is not needed: the workflow runs `buf generate` in `client-rust` and works with either input. S-0008 (the `CARGO_REGISTRY_TOKEN` secret) is needed for a real publish only, not for the test run, so it is not an `after`.

Layers:

| Layer | Tasks |
|-------|-------|
| 1 | T-0061 workflow, T-0062 Cargo.toml |
| 2 | T-0063 README, after the workflow so that it describes what the workflow does |
| 3 | T-0065 test-tag run, after T-0062 and T-0063 |

Assumptions:

- crates.io cannot delete a version, so the test tag must not publish. A tag whose version has a pre-release suffix runs `cargo publish --dry-run` and makes a pre-release GitHub Release. Criterion 1 is met: the trigger stays `client-rust/v*` only, with no `workflow_dispatch`.
- The credential is `CARGO_REGISTRY_TOKEN`, as the runbook and `design/system/release.md` decide; trusted publishing is not used.
- The `version` field in `Cargo.toml` stays a placeholder (`0.6.0` now); the workflow overwrites it from the tag.
- S-0020 also edits `client-rust/README.md` (a different section) and `client-rust/buf.gen.yaml`; the pull order sets who goes first.

Forecast: flai gave 12m (median of 3 done stories, size 6). Adjusted to 2h: four tasks, with the workflow needing care over tag parsing, annotation notes, and `--allow-dirty`, and a real GitHub run that can need a retry. Delivery moves from 2026-10-28T10:42:00Z to 2026-10-28T12:30:00Z with it.

Cost of delay: not set. `flai cod` stopped because neither the story nor E-0002 has the operator's inputs; asked on the plan thread.
