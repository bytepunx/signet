---
id: T-0082
type: task
nature: feature
title: "Write release-client-elixir.yml: test, publish to hex.pm, and create the GitHub Release on client-elixir/v* tags"
status: done
parent: S-0016
owner: alex
created: 2026-10-27T22:52:56Z
updated: 2026-11-06T07:59:17Z
transitions:
  - to: ready
    at: 2026-11-06T07:59:14Z
    by: agent-S-0016
  - to: in-progress
    at: 2026-11-06T07:59:15Z
    by: agent-S-0016
  - to: done
    at: 2026-11-06T07:59:17Z
    by: agent-S-0016
stream: S-0016
tags: [client-elixir]
touches: [".github/workflows/release-client-elixir.yml"]
after: [T-0075]
usage:
  source: log
  seconds: 2
  estimated: true
  priced_by: harness
  models: []
---
# T-0082 Write release-client-elixir.yml: test, publish to hex.pm, and create the GitHub Release on client-elixir/v* tags

## Work

Write `.github/workflows/release-client-elixir.yml`:

- `on: push: tags: ["client-elixir/v*"]` and nothing else; `permissions: contents: write`; a `concurrency` group on the tag so a re-pushed tag does not run twice at once.
- One job on `ubuntu-latest`: `actions/checkout` with `fetch-depth: 0`, then `git fetch --force --tags` so the annotated tag object is present (checkout can turn it into a lightweight ref).
- Derive the version: `VERSION="${GITHUB_REF_NAME#client-elixir/v}"`, failing when it is empty or equals the tag name.
- Run `scripts/test-client-elixir.sh`, then `scripts/publish-client-elixir.sh "$VERSION"` with `HEX_API_KEY: ${{ secrets.HEX_API_KEY }}` in that step's `env` only.
- Create the GitHub Release with `gh release create "$GITHUB_REF_NAME" --title "$GITHUB_REF_NAME" --notes-file notes.md` (a prerelease flag when `VERSION` contains `-`), where `notes.md` is the tag annotation (`git for-each-ref --format='%(contents)' "refs/tags/$GITHUB_REF_NAME"`) followed by the subjects of the commits since the previous `client-elixir/v*` tag that changed `client-elixir/` (`git log --format='- %s' PREV..TAG -- client-elixir`), or since the start of history for the first release. Use `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}`.
- Pin third-party actions to a major version as the repository's `system-flow-check.yml` does.

## Done when

- `actionlint` (or `docker run --rm rhysd/actionlint`) reports nothing on the file, and `flai check --strict` and the markdown lint pass.
- The file triggers on `client-elixir/v*` tags only, runs the test script before the publish script, gives `HEX_API_KEY` only to the publish step, and creates the Release from the annotation.
- Running the notes commands in the shell against a local annotated tag `client-elixir/v0.0.0-test` (deleted afterwards) prints the annotation and the commit list.

## Notes

It waits for the publish script, which it calls. The design is `design/system/release.md` § Release and CI, steps 4 and the Components row for `client-elixir`. A real run needs `HEX_API_KEY` from S-0008 and is the operator's first release, `client-elixir/v0.1.0`; the story does not publish a test version, because a Hex release can be retired but not removed after an hour.
