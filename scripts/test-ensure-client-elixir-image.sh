#!/usr/bin/env sh
# Tests scripts/ensure-client-elixir-image.sh without a Docker daemon: a fake
# `docker` first on PATH answers `docker image inspect` from FAKE_LABEL (empty
# and failing when FAKE_MISSING=1) and records every call. Asserts the image
# build is skipped when the image carries the Dockerfile's current hash, and runs
# when the image is missing, when the label differs, when the image has no
# label, and when SIGNET_REBUILD=1. This is I-0002: the build ran every run.
#
#   scripts/test-ensure-client-elixir-image.sh
set -eu
cd "$(dirname "$0")/.."

root="$(pwd)"
work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT
mkdir "${work}/bin"

cat >"${work}/bin/docker" <<'EOF'
#!/usr/bin/env sh
echo "$*" >>"${FAKE_DOCKER_LOG}"
case "$1 $2" in
  "image inspect")
    [ "${FAKE_MISSING:-}" = "1" ] && exit 1
    echo "${FAKE_LABEL:-}"
    ;;
  build*) ;;
  *) echo "fake docker: unexpected call: $*" >&2; exit 2 ;;
esac
EOF
chmod +x "${work}/bin/docker"

if command -v sha256sum >/dev/null 2>&1; then
  hash="$(sha256sum client-elixir/Dockerfile | cut -d ' ' -f 1)"
else
  hash="$(shasum -a 256 client-elixir/Dockerfile | cut -d ' ' -f 1)"
fi

failures=0

# check <name> <expect: built|skipped> [VAR=value ...] — runs the script with
# only the given variables set and compares what the fake docker saw.
check() {
  name="$1"
  expect="$2"
  shift 2
  : >"${work}/log"
  if ! env -u SIGNET_REBUILD PATH="${work}/bin:${PATH}" FAKE_DOCKER_LOG="${work}/log" "$@" \
    "${root}/scripts/ensure-client-elixir-image.sh" >"${work}/out" 2>&1; then
    echo "FAIL ${name}: script exited non-zero" >&2
    cat "${work}/out" >&2
    failures=$((failures + 1))
    return
  fi
  if grep -q '^build ' "${work}/log"; then got=built; else got=skipped; fi
  if [ "${got}" != "${expect}" ]; then
    echo "FAIL ${name}: expected ${expect}, got ${got}" >&2
    cat "${work}/log" >&2
    failures=$((failures + 1))
    return
  fi
  if [ "${expect}" = "built" ] && ! grep -q "signet.dockerfile-sha256=${hash}" "${work}/log"; then
    echo "FAIL ${name}: build did not label the image with ${hash}" >&2
    failures=$((failures + 1))
    return
  fi
  echo "ok   ${name}"
}

check "label matches: build skipped" skipped FAKE_LABEL="${hash}"
check "image missing: build runs" built FAKE_MISSING=1
check "label differs: build runs" built FAKE_LABEL="0000"
check "image has no label: build runs" built FAKE_LABEL=""
check "SIGNET_REBUILD=1: build runs" built FAKE_LABEL="${hash}" SIGNET_REBUILD=1

if [ "${failures}" -gt 0 ]; then
  echo "test-ensure-client-elixir-image.sh: ${failures} failed" >&2
  exit 1
fi
echo "test-ensure-client-elixir-image.sh: all passed"
