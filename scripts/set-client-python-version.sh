#!/usr/bin/env sh
# Writes a release version into client-python/pyproject.toml, so the version
# step of release-client-python.yml is one named, locally testable command.
# Accepts client-python/v0.6.0, v0.6.0, or 0.6.0, with an optional PEP 440
# pre-release suffix (a1, b2, rc1). Prints only the version it wrote.
#
#   scripts/set-client-python-version.sh client-python/v0.6.0
set -eu
cd "$(dirname "$0")/.."

if [ "$#" -ne 1 ] || [ -z "$1" ]; then
  echo "set-client-python-version.sh: usage: scripts/set-client-python-version.sh <tag-or-version>" >&2
  exit 2
fi
input="$1"
version="${input#client-python/}"
version="${version#v}"

if ! printf '%s\n' "${version}" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+((a|b|rc)[0-9]+)?$'; then
  echo "set-client-python-version.sh: '${input}' is not X.Y.Z with an optional a, b, or rc suffix (PyPI rejects other forms)" >&2
  exit 1
fi

file="client-python/pyproject.toml"
count=$(grep -c '^version = "' "${file}" || true)
if [ "${count}" -ne 1 ]; then
  echo "set-client-python-version.sh: ${file} has ${count} top-level version lines; expected 1" >&2
  exit 1
fi

tmp=$(mktemp)
sed "s/^version = \".*\"/version = \"${version}\"/" "${file}" > "${tmp}"
cat "${tmp}" > "${file}"
rm -f "${tmp}"
echo "${version}"
