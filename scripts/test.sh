#!/usr/bin/env sh
# Behavior tests: fast, no external dependencies beyond Docker. Run on every iteration.
# One line per component; each script exits non-zero on failure. Every component
# runs even when an earlier one fails, so one run shows every failure.
set -eu
cd "$(dirname "$0")/.."

failed=""
echo "test.sh: ensure-client-elixir-image"
if ! scripts/test-ensure-client-elixir-image.sh; then
  echo "test.sh: ensure-client-elixir-image FAILED" >&2
  failed="${failed} ensure-client-elixir-image"
fi
for component in server proto client-go client-python client-typescript client-rust client-csharp client-elixir; do
  echo "test.sh: ${component}"
  if "scripts/test-${component}.sh"; then
    echo "test.sh: ${component} passed"
  else
    echo "test.sh: ${component} FAILED" >&2
    failed="${failed} ${component}"
  fi
done

if [ -n "${failed}" ]; then
  echo "test.sh: failed:${failed}" >&2
  exit 1
fi
echo "test.sh: every component passed"
