---
id: T-0061
type: task
nature: feature
title: "Write release-client-rust.yml: version from the tag, test, publish to crates.io, GitHub Release"
status: done
parent: S-0014
owner: alex
created: 2026-10-27T22:52:11Z
updated: 2026-11-06T07:53:44Z
transitions:
  - to: ready
    at: 2026-11-06T07:53:44Z
    by: agent-S-0014
  - to: in-progress
    at: 2026-11-06T07:53:44Z
    by: agent-S-0014
  - to: done
    at: 2026-11-06T07:53:44Z
    by: agent-S-0014
stream: S-0014
tags: [client-rust]
touches: [".github/workflows/release-client-rust.yml"]
usage:
  source: log
  seconds: 0
  models: []
---
# T-0061 Write release-client-rust.yml: version from the tag, test, publish to crates.io, GitHub Release

## Work

Write `.github/workflows/release-client-rust.yml`, modelled on signet-clients' `publish-rust.yml` (read it with `gh api repos/bytepunx/signet-clients/contents/.github/workflows/publish-rust.yml`) and on the flow in `design/system/release.md`.

- Trigger on `push` of tags `client-rust/v*` only. Permissions `contents: write`; no other trigger, no `workflow_dispatch`.
- Check out with full history and fetch the tags (`fetch-depth: 0`, then `git fetch --force --tags origin`) so the annotation and the previous tag are readable.
- Derive the version: strip `client-rust/v` from `GITHUB_REF_NAME`, fail unless it is a semver (`X.Y.Z` or `X.Y.Z-pre`).
- Write the version into `client-rust/Cargo.toml` (the `[package]` `version` line only) before anything builds, and fail if the line did not change.
- Test as CI does: run `scripts/test-client-rust.sh` (created by S-0007).
- Set up the Rust toolchain and `buf` (`bufbuild/buf-action` with `setup_only`), run `buf generate` in `client-rust`, then `cargo publish --allow-dirty` with `CARGO_REGISTRY_TOKEN` from the repository secret. `--allow-dirty` is needed because `src/gen` is gitignored and `Cargo.toml` was edited; keep the comment explaining it. `Cargo.toml`'s `include` still decides what ships.
- Safe test runs: crates.io cannot delete a published version. When the version has a pre-release suffix (contains `-`), run `cargo publish --dry-run` instead and mark the GitHub Release as a pre-release. Say so in a comment at the step.
- Create the GitHub Release named by the tag with `gh release create`, using the tag annotation (`git for-each-ref --format='%(contents)' refs/tags/<tag>`) followed by the commits since the previous `client-rust/v*` tag (`git log --oneline <prev>..<tag> -- client-rust`; the whole log when there is no previous tag) as the notes.
- Run the publish step before the release step, so a failed publish leaves no Release behind.

## Done when

- The file exists, passes `actionlint` (or a YAML parse if `actionlint` is not installed, stating which) and the markdown and YAML lint the repository runs.
- Reading it shows: the only trigger is `client-rust/v*`; the version is written to `Cargo.toml` before `cargo publish`; a pre-release version only dry-runs; the Release body is the annotation plus the commits since the previous tag.
- `flai check --strict` is clean.

## Notes

Waits for nothing in this story. The story itself follows S-0007, which creates `scripts/test-client-rust.sh`.
