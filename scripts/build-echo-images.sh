#!/usr/bin/env sh
# Build the smoke-test echo image of one or more clients.
#
#   scripts/build-echo-images.sh <language>... | all [--tag <tag>] [--push]
#
# Builds ghcr.io/bytepunx/signet-echo-<language>:<tag> (default tag: local) from
# client-<language>/examples/echo/Dockerfile. The build context comes from the
# `docker build -f ... .` line in each Dockerfile's header comment: a -f path
# that starts with client-<language>/ means the repository root, any other
# means it runs from inside client-<language>/. --push pushes the tag; without
# it nothing leaves the host. The images are test-only: they print secrets.
set -eu
cd "$(dirname "$0")/.."

registry="ghcr.io/bytepunx/signet-echo"
languages="go python typescript rust csharp"
tag="local"
push=0
selected=""

while [ $# -gt 0 ]; do
  case "$1" in
    --tag)
      [ $# -ge 2 ] || { echo "build-echo-images.sh: --tag needs a value" >&2; exit 2; }
      tag="$2"; shift 2 ;;
    --push) push=1; shift ;;
    all) selected="$selected $languages"; shift ;;
    -*) echo "build-echo-images.sh: unknown option $1" >&2; exit 2 ;;
    *) selected="$selected $1"; shift ;;
  esac
done

if [ -z "$selected" ]; then
  echo "usage: build-echo-images.sh <language>... | all [--tag <tag>] [--push]" >&2
  exit 2
fi

# Check every language before building any.
for lang in $selected; do
  dockerfile="client-$lang/examples/echo/Dockerfile"
  case " $languages elixir " in
    *" $lang "*) ;;
    *) echo "build-echo-images.sh: unknown language '$lang' (known: $languages)" >&2; exit 2 ;;
  esac
  if [ ! -f "$dockerfile" ]; then
    echo "build-echo-images.sh: '$lang' has no echo image: $dockerfile does not exist" >&2
    exit 2
  fi
done

for lang in $selected; do
  dockerfile="client-$lang/examples/echo/Dockerfile"
  line=$(grep -E '^#[[:space:]]+docker build -f ' "$dockerfile" | head -n 1 || true)
  if [ -z "$line" ]; then
    echo "build-echo-images.sh: $dockerfile has no 'docker build -f' line in its header" >&2
    exit 1
  fi
  file=$(printf '%s\n' "$line" | sed -E 's/^#[[:space:]]+docker build -f ([^[:space:]]+).*/\1/')
  case "$file" in
    "client-$lang/"*) context="." ; file_arg="$file" ;;
    *) context="client-$lang"; file_arg="client-$lang/$file" ;;
  esac
  image="$registry-$lang:$tag"
  echo "+ docker build -f $file_arg -t $image $context"
  docker build -f "$file_arg" -t "$image" "$context"
  if [ "$push" -eq 1 ]; then
    echo "+ docker push $image"
    docker push "$image"
  fi
done
