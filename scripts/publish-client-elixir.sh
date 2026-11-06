#!/usr/bin/env sh
# Publishes signet_client to hex.pm from the Docker toolchain image, with the
# version taken from the release tag (no commit bumps it). Called by
# .github/workflows/release-client-elixir.yml.
#
#   scripts/publish-client-elixir.sh client-elixir/v0.1.0            # publish
#   scripts/publish-client-elixir.sh --dry-run client-elixir/v0.1.0  # build the package, publish nothing
#
# The tag may be client-elixir/vX.Y.Z, vX.Y.Z, or X.Y.Z. Publishing needs
# HEX_API_KEY in the environment; BUF_TOKEN is passed through when set. The
# dry run needs neither: it generates the stubs, fetches dependencies, and
# runs mix hex.build, which prints the package's version and files.
set -eu
cd "$(dirname "$0")/.."

dry_run=0
if [ "${1:-}" = "--dry-run" ]; then
  dry_run=1
  shift
fi
if [ "$#" -ne 1 ]; then
  echo "usage: publish-client-elixir.sh [--dry-run] <tag>" >&2
  exit 2
fi
version="${1#client-elixir/}"
version="${version#v}"
case "${version}" in
  [0-9]*.[0-9]*.[0-9]*) ;;
  *) echo "publish-client-elixir.sh: '$1' does not name a version" >&2; exit 2 ;;
esac

if [ "${dry_run}" -eq 0 ] && [ -z "${HEX_API_KEY:-}" ]; then
  echo "publish-client-elixir.sh: HEX_API_KEY is not set" >&2
  exit 1
fi

image="signet-client-elixir:toolchain"
echo "publish-client-elixir.sh: building ${image}"
docker build -q -t "${image}" client-elixir >/dev/null

run() {
  docker run --rm \
    -v "$(pwd)/client-elixir:/app" \
    -e MIX_ENV=dev \
    -e SIGNET_CLIENT_VERSION="${version}" \
    -e BUF_TOKEN \
    -e HEX_API_KEY \
    "${image}" "$@"
}

echo "publish-client-elixir.sh: generating stubs"
run buf generate
run buf generate --template buf.gen.spiffe.yaml
echo "publish-client-elixir.sh: fetching dependencies"
run mix deps.get

if [ "${dry_run}" -eq 1 ]; then
  echo "publish-client-elixir.sh: dry run, building signet_client ${version}"
  run mix hex.build
else
  echo "publish-client-elixir.sh: publishing signet_client ${version}"
  run mix hex.publish --yes
fi
