---
id: T-0034
type: task
nature: improvement
title: "Rust client: generate src/gen from ../proto/proto and fix the echo image build"
status: backlog
parent: S-0020
owner: alex
created: 2026-10-27T22:25:40Z
updated: 2026-10-27T22:25:40Z
transitions: []
stream: S-0020
tags: [client-rust, proto]
touches: [client-rust/buf.gen.yaml, client-rust/README.md, client-rust/examples/echo/Dockerfile]
---
# T-0034 Rust client: generate src/gen from ../proto/proto and fix the echo image build

## Work

Replace the pinned BSR module input in `client-rust/buf.gen.yaml` with `directory: ../proto/proto` and rewrite its comment. The echo image has a `protogen` stage that runs `buf generate` from a copy of `buf.gen.yaml` alone, which a relative directory input breaks: change the Dockerfile to build from the repository root, copy `proto/proto` and `client-rust/buf.gen.yaml` to the same relative layout the repository has, and run `buf generate` there; update its build comment to match. In `client-rust/README.md` rewrite the opening paragraph and the regenerate section (about lines 4 and 23 to 29) so they say `buf generate` reads `../proto/proto` and no longer tell readers to bump a BSR pin.

## Done when

- `client-rust/buf.gen.yaml` names `directory: ../proto/proto` as its only input.
- `buf generate` in `client-rust/` writes the same `src/gen` files as the `v0.4.0` pin did.
- `cargo test` in `client-rust/` passes.
- The echo image builds from the repository root with the documented command.
- `client-rust/README.md` no longer tells readers to bump a BSR pin.

## Notes
