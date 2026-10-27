#!/usr/bin/env sh
# Runs `buf generate` for one component inside the pinned bufbuild/buf image,
# so nothing is installed on a host or a runner beyond Docker. The repository
# root is mounted at /repo and buf runs in /repo/<component>, so a buf.gen.yaml
# input of a BSR module or of ../proto/proto both resolve.
#
#   scripts/buf-generate.sh client-python
#   scripts/buf-generate.sh server --template buf.gen.yaml
#
# BUF_TOKEN, when set, is passed through so remote plugins are not rate limited.
set -eu
cd "$(dirname "$0")/.."

image="bufbuild/buf:1.71.0"

if [ "$#" -lt 1 ]; then
  echo "buf-generate.sh: usage: scripts/buf-generate.sh <component> [buf generate args]" >&2
  exit 2
fi
component="$1"
shift

if [ ! -f "${component}/buf.gen.yaml" ]; then
  echo "buf-generate.sh: ${component}/buf.gen.yaml not found; name a component folder that has one (server, client-go, client-python, ...)" >&2
  exit 2
fi

cache="$(pwd)/.cache/buf"
mkdir -p "${cache}"

echo "buf-generate.sh: buf generate in ${component} (${image})"
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -e HOME=/repo/.cache/buf \
  -e BUF_CACHE_DIR=/repo/.cache/buf \
  ${BUF_TOKEN:+-e BUF_TOKEN} \
  -v "$(pwd):/repo" \
  -w "/repo/${component}" \
  "${image}" generate "$@"
