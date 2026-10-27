#!/usr/bin/env sh
# Integration tests: one component against real adapters. Run after behavior tests pass.
# One line per component; each script exits non-zero on failure.
set -eu
cd "$(dirname "$0")/.."
scripts/integration-client-elixir.sh
