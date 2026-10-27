#!/usr/bin/env sh
# Behavior tests for the Rust client, run inside Docker so nothing is installed
# on the host. Generates the stubs into client-rust/src/gen (gitignored) with
# scripts/buf-generate.sh, then with client-rust/ mounted at /app runs
# cargo test for the default features and for spiffe-workload. The cargo
# registry and target/ live in named volumes, so nothing in the checkout is
# root-owned.
#
#   scripts/test-client-rust.sh                  # both cargo test runs
#   scripts/test-client-rust.sh cargo clippy     # any other command
set -eu
cd "$(dirname "$0")/.."

image="rust:1.90-bookworm"

echo "test-client-rust.sh: generating stubs"
scripts/buf-generate.sh client-rust

run() {
  docker run --rm \
    -v "$(pwd)/client-rust:/app" \
    -v signet-client-rust-cargo:/usr/local/cargo/registry \
    -v signet-client-rust-target:/app/target \
    -w /app \
    "${image}" "$@"
}

if [ "$#" -gt 0 ]; then
  echo "test-client-rust.sh: running $*"
  run "$@"
else
  echo "test-client-rust.sh: cargo test --all-targets"
  run cargo test --all-targets
  echo "test-client-rust.sh: cargo test --features spiffe-workload"
  run cargo test --features spiffe-workload
fi
