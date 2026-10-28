#!/bin/sh
# Check or set the Actions secrets the release workflows need.
# Usage: scripts/set-release-secrets.sh [check|set]
#   check  (default) lists each secret as present or missing; non-zero while any is missing
#   set    reads each value from the environment variable of the same name and sets it
# Override the repository with SIGNET_REPO. Values never appear in arguments or output.
set -eu

REPO="${SIGNET_REPO:-bytepunx/signet}"
SECRETS="BUF_TOKEN DOCKERHUB_USERNAME DOCKERHUB_TOKEN CARGO_REGISTRY_TOKEN NUGET_USER HEX_API_KEY"

mode="${1:-check}"

if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh is not installed; see https://cli.github.com" >&2
  exit 1
fi
if ! gh auth status >/dev/null 2>&1; then
  echo "error: gh is not logged in; run 'gh auth login'" >&2
  exit 1
fi

case "$mode" in
  check)
    echo "checking secrets of $REPO"
    present=$(gh secret list --repo "$REPO" --json name --jq '.[].name')
    missing=0
    for name in $SECRETS; do
      if printf '%s\n' "$present" | grep -qx "$name"; then
        echo "present  $name"
      else
        echo "missing  $name"
        missing=1
      fi
    done
    exit "$missing"
    ;;
  set)
    echo "setting secrets of $REPO from the environment"
    for name in $SECRETS; do
      value=$(printenv "$name" || true)
      if [ -z "$value" ]; then
        echo "skipped  $name (\$$name is empty)"
        continue
      fi
      printf '%s' "$value" | gh secret set "$name" --repo "$REPO" --body - >/dev/null
      echo "set      $name"
    done
    ;;
  *)
    echo "usage: $0 [check|set]" >&2
    exit 2
    ;;
esac
