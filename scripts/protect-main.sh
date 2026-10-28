#!/bin/sh
# Set branch protection on main with the CI jobs as required status checks.
# Usage: scripts/protect-main.sh [--apply | --show]
#   (none)   dry run: prints the request body and the command
#   --apply  sends the request
#   --show   prints the current protection
# Override the repository with SIGNET_REPO.
#
# Operator's choice (S-0008, TH-0003): acceptance is local and `flai release --pending`
# pushes main directly, so no pull request review is required, enforce_admins is off, and
# strict (up to date) is off so a branch rebased by `flai stream sync` need not re-run CI.
set -eu

REPO="${SIGNET_REPO:-bytepunx/signet}"
# Job names of .github/workflows/ci.yml, plus `check` of system-flow-check.yml.
CHECKS="server proto client-go client-python client-typescript client-rust client-csharp client-elixir check"

if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh is not installed; see https://cli.github.com" >&2
  exit 1
fi
if ! gh auth status >/dev/null 2>&1; then
  echo "error: gh is not logged in; run 'gh auth login'" >&2
  exit 1
fi

endpoint="repos/$REPO/branches/main/protection"

if [ "${1:-}" = "--show" ]; then
  gh api "$endpoint"
  exit 0
fi

contexts=""
for c in $CHECKS; do
  contexts="${contexts:+$contexts, }\"$c\""
done
body=$(cat <<JSON
{
  "required_status_checks": {"strict": false, "contexts": [$contexts]},
  "enforce_admins": false,
  "required_pull_request_reviews": null,
  "restrictions": null
}
JSON
)

echo "branch protection for main of $REPO:"
echo "$body"
case "${1:-}" in
  --apply)
    printf '%s\n' "$body" | gh api -X PUT "$endpoint" --input - >/dev/null
    echo "applied"
    ;;
  "")
    echo "dry run; would run: gh api -X PUT $endpoint --input -"
    echo "rerun with --apply to change it"
    ;;
  *)
    echo "usage: $0 [--apply | --show]" >&2
    exit 2
    ;;
esac
