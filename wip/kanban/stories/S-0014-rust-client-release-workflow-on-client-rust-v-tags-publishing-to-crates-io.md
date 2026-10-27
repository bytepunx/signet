---
id: S-0014
type: story
nature: feature
title: "Rust client: release workflow on client-rust/v* tags publishing to crates.io"
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-10-27T19:48:21Z
transitions: []
tags: [client-rust]
topics: [release, github]
touches: [".github/workflows/release-client-rust.yml", client-rust/Cargo.toml, client-rust/README.md]
after: [S-0007]
---
# S-0014 Rust client: release workflow on client-rust/v* tags publishing to crates.io

## Goal

A push of `client-rust/vX.Y.Z` sets the crate version from the tag, builds, tests, publishes `signet-client` to crates.io, and creates the GitHub Release. `Cargo.toml` names this repository.

## Acceptance criteria
- [ ] `release-client-rust.yml` triggers on `client-rust/v*` only, sets the version in `Cargo.toml` from the tag before `cargo publish`, publishes with `CARGO_REGISTRY_TOKEN` or crates.io trusted publishing, and creates the GitHub Release with notes from the tag annotation
- [ ] `Cargo.toml` repository and documentation fields name `bytepunx/signet`
- [ ] `client-rust/README.md` names the tag form and the workflow

## Tasks

## Notes

Reference: signet-clients' `publish-rust.yml` on GitHub.
