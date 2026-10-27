#!/usr/bin/env sh
# Behavior tests, lint, build, and stub-drift check for the server, run inside
# Docker so nothing is installed on the host. With server/ mounted at /app and
# Go module and build caches kept in named volumes between runs:
#   1. go test -race (the Debian golang image: -race needs CGO)
#   2. golangci-lint, pinned to v1 because server/.golangci.yml is the v1 schema
#      and server/githooks/pre-commit pins the same version; installed with the
#      Go image's own toolchain, since a prebuilt v1 binary refuses a go.mod
#      newer than the Go it was built with
#   3. make build
#   4. stub drift: regenerate server/gen and fail on any difference
#   5. Helm chart lint and render checks (scripts/check-server-chart.sh)
#
#   scripts/test-server.sh
set -eu
cd "$(dirname "$0")/.."

go_image="golang:1.26"
lint_version="v1.64.8"

go_run() {
  docker run --rm \
    -v "$(pwd)/server:/app" \
    -v signet-server-gomod:/go/pkg/mod \
    -v signet-server-gobuild:/root/.cache/go-build \
    -v signet-server-gobin:/go/bin \
    -w /app \
    "$@"
}

echo "test-server.sh: [1/5] go test -race"
go_run "${go_image}" go test -race -count=1 ./...

echo "test-server.sh: [2/5] golangci-lint ${lint_version}"
go_run "${go_image}" sh -c "
  [ \"\$(golangci-lint version --format short 2>/dev/null)\" = \"${lint_version#v}\" ] ||
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@${lint_version}
  golangci-lint run ./..."

echo "test-server.sh: [3/5] make build"
go_run "${go_image}" make build

echo "test-server.sh: [4/5] stub drift"
scripts/buf-generate.sh server
if ! git diff --quiet -- server/gen; then
  git diff -- server/gen
  echo "test-server.sh: server/gen is out of sync with its schema; run scripts/buf-generate.sh server and commit the result" >&2
  exit 1
fi

echo "test-server.sh: [5/5] Helm chart checks"
scripts/check-server-chart.sh

echo "test-server.sh: passed"
