#!/usr/bin/env sh
# Ensures the Elixir client's toolchain image exists and matches
# client-elixir/Dockerfile, building it only when it must. The image carries the
# label signet.dockerfile-sha256 with the SHA-256 of the Dockerfile it was built
# from; the build is skipped when the image exists and the label equals the
# Dockerfile's current hash. Checking the image talks only to the local Docker
# daemon, so a skipped build never resolves the base image on a registry.
# Set SIGNET_REBUILD=1 to build regardless.
#
#   scripts/ensure-client-elixir-image.sh
#   SIGNET_REBUILD=1 scripts/ensure-client-elixir-image.sh
set -eu
cd "$(dirname "$0")/.."

image="signet-client-elixir:toolchain"
dockerfile="client-elixir/Dockerfile"
label="signet.dockerfile-sha256"

if command -v sha256sum >/dev/null 2>&1; then
  hash="$(sha256sum "${dockerfile}" | cut -d ' ' -f 1)"
else
  hash="$(shasum -a 256 "${dockerfile}" | cut -d ' ' -f 1)"
fi

current="$(docker image inspect --format "{{ index .Config.Labels \"${label}\" }}" "${image}" 2>/dev/null || true)"

if [ "${SIGNET_REBUILD:-}" = "1" ]; then
  reason="SIGNET_REBUILD=1"
elif [ -z "${current}" ]; then
  reason="no ${image} image with a ${label} label"
elif [ "${current}" != "${hash}" ]; then
  reason="${dockerfile} changed since ${image} was built"
else
  echo "ensure-client-elixir-image.sh: skipped build, ${image} matches ${dockerfile}"
  exit 0
fi

echo "ensure-client-elixir-image.sh: building ${image} (${reason})"
docker build -q --label "${label}=${hash}" -t "${image}" client-elixir >/dev/null
echo "ensure-client-elixir-image.sh: built ${image}"
