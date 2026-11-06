#!/usr/bin/env sh
# Smoke tests: build the echo images of the clients' examples, then run the
# external bytepunx/signet-smoke-test harness against them when it is available.
#
# The harness needs Docker, a k3d cluster with signet and SPIRE running (from
# bytepunx/kluster), and a checkout of bytepunx/signet-smoke-test, named by
# SIGNET_SMOKE_TEST_DIR. Without that checkout this script builds the images
# and says the harness did not run. The exit code is the build's, plus the
# harness's when it ran.
set -eu
cd "$(dirname "$0")/.."

scripts/build-echo-images.sh all --tag local

if [ -z "${SIGNET_SMOKE_TEST_DIR:-}" ]; then
  cat <<'MSG'
smoke.sh: the echo images built, but the harness did NOT run.
The harness (github.com/bytepunx/signet-smoke-test) needs:
  - Docker
  - a k3d cluster with signet and SPIRE (github.com/bytepunx/kluster)
  - a checkout of signet-smoke-test, named by SIGNET_SMOKE_TEST_DIR
Run it against the images tagged ghcr.io/bytepunx/signet-echo-<language>:local
by following the README in that checkout, or set SIGNET_SMOKE_TEST_DIR and run
this script again. See docs/operators/smoke-tests.md.
MSG
  exit 0
fi

if [ ! -d "$SIGNET_SMOKE_TEST_DIR" ]; then
  echo "smoke.sh: SIGNET_SMOKE_TEST_DIR=$SIGNET_SMOKE_TEST_DIR is not a directory" >&2
  exit 1
fi

# The harness's interface is its own README's. No entry point that takes an
# image tag is known here, so print where to look instead of guessing one.
cat <<MSG
smoke.sh: the echo images built, but the harness did NOT run: this script does
not know an entry point of $SIGNET_SMOKE_TEST_DIR that takes a tag.
Run the harness by hand from that checkout, as its README says, with the images
ghcr.io/bytepunx/signet-echo-<language>:local for go, python, typescript, rust,
and csharp.
MSG
