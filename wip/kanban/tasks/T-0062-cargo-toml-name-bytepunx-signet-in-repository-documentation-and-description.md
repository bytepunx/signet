---
id: T-0062
type: task
nature: improvement
title: "Cargo.toml: name bytepunx/signet in repository, documentation, and description"
status: in-progress
parent: S-0014
owner: alex
created: 2026-10-27T22:52:14Z
updated: 2026-11-06T07:53:44Z
transitions:
  - to: ready
    at: 2026-11-06T07:53:44Z
    by: agent-S-0014
  - to: in-progress
    at: 2026-11-06T07:53:44Z
    by: agent-S-0014
stream: S-0014
tags: [client-rust]
touches: [client-rust/Cargo.toml]
---
# T-0062 Cargo.toml: name bytepunx/signet in repository, documentation, and description

## Work

In `client-rust/Cargo.toml`:

- Set `repository = "https://github.com/bytepunx/signet"`.
- Add `documentation = "https://docs.rs/signet-client"`, the docs.rs page of the crate.
- Change `description` from "generated from bytepunx/signet-proto" to say it is generated from the `proto/` directory of bytepunx/signet.
- Update the comments that name `bytepunx/signet-proto` or `../.gitignore` so they say the stubs are regenerated with `buf generate` before packaging. Leave `include`, `version`, and every dependency as they are: the version is a placeholder that the release workflow overwrites.

This task does not wait on the workflow: no field depends on it.

## Done when

- `repository` and `documentation` in `client-rust/Cargo.toml` name `bytepunx/signet` (`documentation` through docs.rs), and no field or comment there still names `signet-clients`.
- `version` and `include` are unchanged.
- `cargo metadata --no-deps --format-version 1 --manifest-path client-rust/Cargo.toml` parses the file (in Docker with a Rust image, or `cargo` on the host if present; say which).
- `flai check --strict` is clean.

## Notes

Runs in the same layer as the workflow task: the two share no path.
