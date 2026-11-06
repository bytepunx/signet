---
id: T-0063
type: task
nature: improvement
title: "client-rust README: name the client-rust/v tag form and the release workflow"
status: done
parent: S-0014
owner: alex
created: 2026-10-27T22:52:20Z
updated: 2026-11-06T07:53:47Z
transitions:
  - to: ready
    at: 2026-11-06T07:53:44Z
    by: agent-S-0014
  - to: in-progress
    at: 2026-11-06T07:53:44Z
    by: agent-S-0014
  - to: done
    at: 2026-11-06T07:53:47Z
    by: agent-S-0014
stream: S-0014
tags: [client-rust]
touches: [client-rust/README.md]
after: [T-0061]
usage:
  source: log
  seconds: 3
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 2
      output: 696
      cache_read: 62414
      cache_write: 7556
      cost: 0.0434
---
# T-0063 client-rust README: name the client-rust/v tag form and the release workflow

## Work

In `client-rust/README.md`, rewrite the paragraph under `## Install` that says `publish-rust.yml` publishes on a release-please `rust-v*` tag. It must say:

- the crate is published to crates.io when a tag `client-rust/vX.Y.Z` is pushed, by `.github/workflows/release-client-rust.yml` (link it with a relative path);
- the version in `Cargo.toml` is a placeholder: the workflow writes the tag's version before it publishes, and the tag is the version;
- a tag with a pre-release suffix such as `client-rust/v0.0.0-test` only runs `cargo publish --dry-run`;
- operators cut the tag with `flai release --pending`, linking `../docs/operators/release.md`;
- the GitHub Release carries the notes, and `CHANGELOG.md` is history from the old release-please setup.

Leave the `## Regenerating stubs` section alone: S-0020 rewrites it. Fix only links in `## Install` and nothing else in the file. This task follows the workflow task so that the README describes what the workflow does.

## Done when

- `client-rust/README.md` contains the tag form `client-rust/vX.Y.Z` and the filename `release-client-rust.yml`, and no longer mentions `publish-rust.yml` or `rust-v*`.
- The markdown lint (`scripts/lint-md.sh`) and `flai check --strict` are clean.

## Notes

S-0020 also edits this file (the stubs section). The two touch different lines; the pull order decides which goes first.
