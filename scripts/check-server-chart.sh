#!/usr/bin/env sh
# Helm lint and render checks for the server chart, run inside Docker so Helm
# is not installed on the host. Called by scripts/test-server.sh; also runnable
# on its own:
#   1. helm lint
#   2. helm template validates the values schema
#   3. admin.clusterAccess off (default): loopback admin address, no admin port
#   4. admin.clusterAccess on: admin address rebound, admin port present
#   5. admin.clusterAccess on without admin.tls: the render fails
set -eu
cd "$(dirname "$0")/.."

image="alpine/helm:3.17.0"
base="--set signet.trustDomain=example.org --set signet.existingSecret=signet-secrets"

helm() {
  docker run --rm \
    -v "$(pwd)/server/deploy/helm/signet:/chart:ro" \
    "${image}" "$@"
}

fail() {
  echo "check-server-chart.sh: $*" >&2
  exit 1
}

echo "check-server-chart.sh: [1/5] helm lint"
# shellcheck disable=SC2086
helm lint /chart ${base}

echo "check-server-chart.sh: [2/5] helm template validates the values schema"
# shellcheck disable=SC2086
helm template signet /chart ${base} >/dev/null

echo "check-server-chart.sh: [3/5] admin.clusterAccess off omits the admin port"
# shellcheck disable=SC2086
out="$(helm template signet /chart ${base})"
echo "${out}" | grep -q '"127.0.0.1:8444"' ||
  fail "expected a loopback-only SIGNET_ADMIN_ADDR by default"
if echo "${out}" | grep -q 'name: admin'; then
  fail "did not expect an admin-named port when admin.clusterAccess is off"
fi

echo "check-server-chart.sh: [4/5] admin.clusterAccess on adds the admin port"
# shellcheck disable=SC2086
out="$(helm template signet /chart ${base} \
  --set admin.clusterAccess=true --set admin.tls.acknowledgeInsecure=true)"
echo "${out}" | grep -q '"0.0.0.0:8444"' ||
  fail "expected SIGNET_ADMIN_ADDR to be rebound off loopback"
echo "${out}" | grep -q 'name: admin' ||
  fail "expected an admin-named port in the rendered manifests"

echo "check-server-chart.sh: [5/5] admin.clusterAccess on without admin.tls fails to render"
# shellcheck disable=SC2086
if helm template signet /chart ${base} --set admin.clusterAccess=true >/dev/null 2>&1; then
  fail "expected the render to fail without admin.tls"
fi

echo "check-server-chart.sh: passed"
