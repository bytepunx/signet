#!/bin/sh
# Point the old repositories at this one, then archive them.
# Usage: scripts/archive-old-repos.sh [--apply]
#   (none)   dry run: prints each step
#   --apply  updates each README, then archives the repository
# Run it only after a release has published from bytepunx/signet.
set -eu

OLD_REPOS="bytepunx/signet-clients bytepunx/signet-proto bytepunx/signet-admin"
NEW_REPO="${SIGNET_REPO:-bytepunx/signet}"

if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh is not installed; see https://cli.github.com" >&2
  exit 1
fi
if ! gh auth status >/dev/null 2>&1; then
  echo "error: gh is not logged in; run 'gh auth login'" >&2
  exit 1
fi

apply=0
case "${1:-}" in
  --apply) apply=1 ;;
  "") ;;
  *) echo "usage: $0 [--apply]" >&2; exit 2 ;;
esac

readme() {
  cat <<TEXT
# $1 (archived)

This repository is archived. Its code now lives in the mono-repo
[$NEW_REPO](https://github.com/$NEW_REPO), under the folders \`client-*\`, \`proto\`, and
\`admin\`.

Releases now come from there as \`<component>/vX.Y.Z\` tags, such as
\`client-go/v0.7.0\` or \`proto/v0.4.0\`.
TEXT
}

for repo in $OLD_REPOS; do
  echo "== $repo"
  if [ "$(gh repo view "$repo" --json isArchived --jq .isArchived)" = "true" ]; then
    echo "already archived; skipped"
    continue
  fi
  if [ "$apply" -eq 0 ]; then
    echo "would replace README.md with a pointer to $NEW_REPO"
    echo "would run: gh repo archive $repo --yes"
    continue
  fi
  sha=$(gh api "repos/$repo/contents/README.md" --jq .sha)
  content=$(readme "$repo" | base64 | tr -d '\n')
  if ! gh api -X PUT "repos/$repo/contents/README.md" \
    -f message="docs: point to $NEW_REPO and archive" \
    -f content="$content" -f sha="$sha" >/dev/null; then
    echo "error: README update failed for $repo; not archiving it" >&2
    exit 1
  fi
  echo "README updated"
  gh repo archive "$repo" --yes
  echo "archived"
done
[ "$apply" -eq 1 ] || echo "dry run; rerun with --apply to change anything"
