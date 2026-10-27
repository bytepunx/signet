#!/usr/bin/env sh
# Behavior tests for the Python client, run inside Docker so nothing is
# installed on the host. Generates the stubs with scripts/buf-generate.sh
# (gitignored under client-python/src/signet and src/admin), then with
# client-python/ mounted at /app in python:3.10, the lowest version
# pyproject.toml supports: creates a virtualenv outside the mount, installs the
# package with its dev and workload extras, and runs pytest. The pip cache
# lives in the named volume signet-client-python-pip.
#
#   scripts/test-client-python.sh                 # pytest
#   scripts/test-client-python.sh python -m pytest -k watch   # any other command
set -eu
cd "$(dirname "$0")/.."

image="python:3.10-slim"

echo "test-client-python.sh: generating stubs"
scripts/buf-generate.sh client-python

if [ "$#" -gt 0 ]; then
  cmd="$*"
else
  cmd="pytest"
fi

echo "test-client-python.sh: installing and running ${cmd}"
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v "$(pwd)/client-python:/app" \
  -v signet-client-python-pip:/pip-cache \
  -w /app \
  -e HOME=/tmp \
  -e PIP_CACHE_DIR=/pip-cache \
  -e PIP_DISABLE_PIP_VERSION_CHECK=1 \
  -e PYTHONDONTWRITEBYTECODE=1 \
  -e PYTEST_ADDOPTS="-p no:cacheprovider" \
  "${image}" sh -euc '
    python -m venv /tmp/venv
    . /tmp/venv/bin/activate
    pip install -q -e ".[dev,workload]"
    exec "$@"
  ' sh ${cmd}
