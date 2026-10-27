#!/usr/bin/env sh
# Behavior tests: fast, no external dependencies beyond Docker. Run on every iteration.
# One line per component; each script exits non-zero on failure.
set -eu
cd "$(dirname "$0")/.."
scripts/test-client-elixir.sh
