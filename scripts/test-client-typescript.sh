#!/usr/bin/env sh
# Behavior tests for the TypeScript client, run inside Docker so nothing is
# installed on the host. Regenerates the stubs with scripts/buf-generate.sh and
# fails if client-typescript/src/gen differs from what is committed, then with
# client-typescript/ mounted at /app in a pinned node image runs npm ci,
# npm run build, and npm test. node_modules/ and dist/ stay on the host,
# gitignored and owned by the caller.
#
#   scripts/test-client-typescript.sh              # npm test
#   scripts/test-client-typescript.sh npm run build # any other command
set -eu
cd "$(dirname "$0")/.."

image="node:24-bookworm"

run() {
  docker run --rm \
    --user "$(id -u):$(id -g)" \
    -e HOME=/tmp \
    -v "$(pwd)/client-typescript:/app" \
    -w /app \
    "${image}" "$@"
}

echo "test-client-typescript.sh: generating stubs"
scripts/buf-generate.sh client-typescript

if ! git diff --quiet -- client-typescript/src/gen; then
  echo "test-client-typescript.sh: client-typescript/src/gen is out of sync with a fresh buf generate; run scripts/buf-generate.sh client-typescript and commit the result" >&2
  git diff -- client-typescript/src/gen >&2
  exit 1
fi

echo "test-client-typescript.sh: installing dependencies"
run npm ci

echo "test-client-typescript.sh: building"
run npm run build

if [ "$#" -gt 0 ]; then
  echo "test-client-typescript.sh: running $*"
  run "$@"
else
  echo "test-client-typescript.sh: testing"
  run npm test
fi
