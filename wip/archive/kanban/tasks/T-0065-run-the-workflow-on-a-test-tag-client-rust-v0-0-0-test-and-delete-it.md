---
id: T-0065
type: task
nature: feature
title: Run the workflow on a test tag client-rust/v0.0.0-test and delete it
status: cancelled
parent: S-0014
owner: alex
created: 2026-10-27T22:52:25Z
updated: 2026-11-06T08:11:41Z
transitions:
  - to: cancelled
    at: 2026-11-06T08:11:41Z
    by: alex
stream: S-0014
tags: [client-rust]
after: [T-0062, T-0063]
---
# T-0065 Run the workflow on a test tag client-rust/v0.0.0-test and delete it

## Work

Prove the workflow end to end without publishing to crates.io, where a version cannot be deleted.

- Push the story branch, tag its head `client-rust/v0.0.0-test` (annotated, with a one-line message), and push the tag.
- Follow the run with `gh run list` and `gh run watch`. It must pass: the version written, `scripts/test-client-rust.sh` green, `buf generate` ok, `cargo publish --dry-run` ok, and a pre-release GitHub Release with the annotation and the commits.
- If the run fails, fix the workflow, push, move the test tag (this tag only, never a real one), and run again.
- Delete the test GitHub Release (`gh release delete client-rust/v0.0.0-test --cleanup-tag --yes`), which removes the tag on the remote, and delete the local tag.
- If a tag cannot be pushed from here, or the repository's Actions are off, stop, record what is missing under `## Open questions`, and leave this task unclosed rather than ticking it.
- Record the run's URL in the story's narrative.

## Done when

- A run of `release-client-rust.yml` on `client-rust/v0.0.0-test` passed, recorded with its URL in the narrative.
- No `client-rust/v0.0.0-test` tag or Release remains on the remote or locally, and `signet-client` `0.0.0-test` is not on crates.io.
- `flai check --strict` is clean.

## Notes

Waits for the Cargo.toml task and the README task. It touches no file. The real publish with `CARGO_REGISTRY_TOKEN` is the operator's first `flai release --pending`; the secret is set by S-0008.
- 2026-11-06T08:11:41Z: moved to cancelled: Skipped
