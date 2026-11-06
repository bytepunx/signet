#!/usr/bin/env sh
# Behavior tests for the Elixir client, run inside Docker so nothing is
# installed on the host. Ensures the toolchain image from client-elixir/Dockerfile
# exists (scripts/ensure-client-elixir-image.sh builds it only when it is missing
# or the Dockerfile changed; SIGNET_REBUILD=1 forces a build), then with
# the repository root mounted at /repo and client-elixir/ as the working directory
# (buf.gen.yaml reads ../proto/proto):
# generates the stubs, fetches dependencies, compiles with warnings as errors,
# and runs mix test. deps/, _build/, and lib/gen/ stay on the host, gitignored.
#
#   scripts/test-client-elixir.sh            # mix test
#   scripts/test-client-elixir.sh mix format # any other mix or buf command
set -eu
cd "$(dirname "$0")/.."

image="signet-client-elixir:toolchain"
scripts/ensure-client-elixir-image.sh

run() {
  docker run --rm \
    -v "$(pwd):/repo" \
    -w /repo/client-elixir \
    -e MIX_ENV="${MIX_ENV:-test}" \
    "${image}" "$@"
}

echo "test-client-elixir.sh: generating stubs"
run buf generate
run buf generate --template buf.gen.spiffe.yaml
echo "test-client-elixir.sh: fetching dependencies"
run mix deps.get

if [ "$#" -gt 0 ]; then
  echo "test-client-elixir.sh: running $*"
  run "$@"
else
  echo "test-client-elixir.sh: compiling and testing"
  run mix do compile --warnings-as-errors + test
fi
