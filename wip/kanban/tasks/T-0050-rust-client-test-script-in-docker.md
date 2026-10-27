---
id: T-0050
type: task
nature: feature
title: Rust client test script in Docker
status: backlog
parent: S-0007
owner: alex
created: 2026-10-27T22:26:31Z
updated: 2026-10-27T22:26:31Z
transitions: []
stream: S-0007
tags: [client-rust, ci]
touches: [scripts/test-client-rust.sh]
after: [T-0042]
---
# T-0050 Rust client test script in Docker

## Work

Add `scripts/test-client-rust.sh`, POSIX `sh` under `set -eu`, modelled on `scripts/test-client-elixir.sh`: run `scripts/buf-generate.sh client-rust` (stubs under `src/gen/` are gitignored), then with `client-rust/` mounted in a pinned `rust` image run `cargo test --all-targets` and `cargo test --features spiffe-workload`, keeping `CARGO_HOME` and `target/` out of root-owned paths (a named volume for the registry cache). The `protoc` that `tonic-prost` or `prost` needs must come from the image or a pinned install inside the container. Extra arguments run instead of the defaults.

It waits for `scripts/buf-generate.sh` (T-0042).

## Done when

- `scripts/test-client-rust.sh` exits 0 on a clean checkout and runs the unit tests in `client-rust/src/`.
- A failing test makes it exit non-zero.
- Nothing but Docker is installed on the host, and `sh -n` is clean.

## Notes

Layer 2 of the plan.
