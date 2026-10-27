#!/usr/bin/env sh
# Behavior tests for the proto module, run inside Docker so nothing is installed
# on the host: buf lint, then buf breaking against the last proto/v* tag. With
# no such tag (none created yet, or tags not fetched) the breaking check is
# skipped and says so. CI checks out with fetch-depth: 0 so the tags exist.
#
#   scripts/test-proto.sh
set -eu
cd "$(dirname "$0")/.."

image="bufbuild/buf:1.71.0"
root="$(pwd)"
# The common git directory holds the tags; in a worktree .git is only a pointer file.
gitdir="$(cd "$(git rev-parse --git-common-dir)" && pwd)"
cache="${TMPDIR:-/tmp}/signet-buf-cache"
mkdir -p "${cache}"

run() {
  docker run --rm \
    --user "$(id -u):$(id -g)" \
    -e HOME=/cache -e BUF_CACHE_DIR=/cache \
    -v "${cache}:/cache" \
    -v "${root}:/repo" \
    -v "${gitdir}:/gitdir:ro" \
    -w /repo/proto \
    "${image}" "$@"
}

echo "test-proto.sh: buf lint"
run lint

tag="$(git tag --list 'proto/v*' --sort=-v:refname | head -n 1)"
if [ -z "${tag}" ]; then
  echo "test-proto.sh: no proto/v* tag found; skipping buf breaking"
  exit 0
fi
echo "test-proto.sh: buf breaking against ${tag}"
run breaking --against "/gitdir#format=git,tag=${tag},subdir=proto"
