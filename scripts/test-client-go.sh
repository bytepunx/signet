#!/usr/bin/env sh
# Behavior tests for the Go client, run inside Docker so nothing is installed on
# the host. Regenerates the stubs with scripts/buf-generate.sh and fails if
# client-go/gen differs from what is committed, then, with client-go/ mounted at
# /app in the golang image go.mod names, runs go vet and go test -race. The Go
# module and build caches live in named volumes between runs.
#
#   scripts/test-client-go.sh                  # go vet, then go test -race
#   scripts/test-client-go.sh go test -run X ./...   # any other command
set -eu
cd "$(dirname "$0")/.."

image="golang:1.26"

echo "test-client-go.sh: generating stubs"
scripts/buf-generate.sh client-go
if ! git diff --quiet -- client-go/gen; then
  echo "test-client-go.sh: client-go/gen is out of sync with buf.gen.yaml; run scripts/buf-generate.sh client-go and commit the result" >&2
  git diff -- client-go/gen >&2
  exit 1
fi

run() {
  docker run --rm \
    -v "$(pwd)/client-go:/app" \
    -v signet-client-go-mod:/go/pkg/mod \
    -v signet-client-go-build:/root/.cache/go-build \
    -w /app \
    "${image}" "$@"
}

if [ "$#" -gt 0 ]; then
  echo "test-client-go.sh: running $*"
  run "$@"
else
  echo "test-client-go.sh: go vet"
  run go vet ./...
  echo "test-client-go.sh: go test"
  run go test -race -count=1 ./...
fi
