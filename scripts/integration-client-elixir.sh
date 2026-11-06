#!/usr/bin/env sh
# Integration tests for the Elixir client, run inside Docker so nothing is
# installed on the host. Ensures the same toolchain image as
# scripts/test-client-elixir.sh (client-elixir/Dockerfile, which also installs
# the real sops binary; built only when missing or changed, see
# scripts/ensure-client-elixir-image.sh), then with client-elixir/ mounted at /app: generates
# the stubs, fetches dependencies, and runs the tests under
# client-elixir/tests/integration/ with the :integration tag included, which
# mix test alone excludes.
#
#   scripts/integration-client-elixir.sh
set -eu
cd "$(dirname "$0")/.."

image="signet-client-elixir:toolchain"
scripts/ensure-client-elixir-image.sh

run() {
  docker run --rm \
    -v "$(pwd)/client-elixir:/app" \
    -e MIX_ENV="${MIX_ENV:-test}" \
    "${image}" "$@"
}

echo "integration-client-elixir.sh: generating stubs"
run buf generate
run buf generate --template buf.gen.spiffe.yaml
echo "integration-client-elixir.sh: fetching dependencies"
run mix deps.get

echo "integration-client-elixir.sh: running tests/integration"
run mix test tests/integration --include integration
