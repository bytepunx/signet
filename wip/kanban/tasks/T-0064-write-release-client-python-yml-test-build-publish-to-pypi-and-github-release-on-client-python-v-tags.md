---
id: T-0064
type: task
nature: feature
title: "Write release-client-python.yml: test, build, publish to PyPI, and GitHub Release on client-python/v* tags"
status: backlog
parent: S-0012
owner: alex
created: 2026-10-27T22:52:24Z
updated: 2026-10-27T22:52:24Z
transitions: []
stream: S-0012
tags: [client-python]
touches: [".github/workflows/release-client-python.yml"]
after: [T-0059, T-0060]
---
# T-0064 Write release-client-python.yml: test, build, publish to PyPI, and GitHub Release on client-python/v* tags

## Work

Add `.github/workflows/release-client-python.yml`. It waits for T-0059 and T-0060 because it calls the version script and builds the package whose `version` placeholder and URLs T-0059 sets. Reference: the old `publish-python.yml` of `bytepunx/signet-clients` (`gh api repos/bytepunx/signet-clients/contents/.github/workflows/publish-python.yml`).

- Trigger: `on: push: tags: ["client-python/v*"]` and nothing else, and top-level `permissions: contents: read`.
- Job `test`: checkout, then `scripts/test-client-python.sh`, the script `ci.yml` runs (S-0007 T-0048). If its name or arguments differ when this is written, follow the script as merged.
- Job `build`, `needs: test`: checkout with `fetch-depth: 0`; set up Python 3.12; set up `buf` with `bufbuild/buf-action` in `setup_only` mode, with the token the old workflow used only if the stubs still come from the BSR (S-0020 may have moved them to `../proto/proto`; follow the `buf.gen.yaml` as merged); `buf generate` in `client-python`; `version=$(scripts/set-client-python-version.sh "$GITHUB_REF_NAME")`; `pip install build`; `python -m build` in `client-python`; check with `python -m twine check client-python/dist/*`; upload `client-python/dist` as an artifact. Verify before it uploads that the wheel contains `signet/v1`, `admin/v1`, and `signet_client`, because the stubs are gitignored and a wheel without them installs but cannot import.
- Job `publish`, `needs: build`: download the artifact and publish with `pypa/gh-action-pypi-publish@release/v1` and `packages-dir` set to the downloaded directory. Permissions are `id-token: write` only. Declare no `environment`, since the trusted publisher that `docs/operators/release.md` has the operator register names the repository and the workflow filename only; say so in a comment, as the old workflow did.
- Job `github-release`, `needs: publish`, `permissions: contents: write`: checkout with `fetch-depth: 0` and `fetch-tags`; read the tag annotation with `git for-each-ref --format='%(contents)' "refs/tags/$GITHUB_REF_NAME"`; find the previous `client-python/v*` tag with `git describe --tags --abbrev=0 --match 'client-python/v*' "$GITHUB_REF_NAME^"` (none for the first release); list the commits since it with `git log --format='- %s (%h)' <previous>..<tag> -- client-python`; write annotation then the list to a notes file; run `gh release create "$GITHUB_REF_NAME" --title "$GITHUB_REF_NAME" --notes-file <file> client-python/dist/*`, with `--prerelease` when the version has an `a`, `b`, or `rc` suffix.
- Keep the shell in the workflow short; anything longer than a few commands goes in a script under `scripts/` (name it in this task's commit if you add one, and add it to the task's touches with `flai touches`).
- Pin third-party actions to a major version or a commit as the other release workflows do. Run `actionlint` if it is available; otherwise say in the commit that it was not.

## Done when

- The workflow triggers on `client-python/v*` tags only, and `actionlint` (or a YAML parse, if actionlint is missing) reports no error.
- A tag `client-python/v0.6.0` would build `signet-client` 0.6.0: running steps 1 to 5 of the `build` job locally in Docker on a scratch copy yields `signet_client-0.6.0-py3-none-any.whl` and a `.tar.gz` of the same version, and the wheel lists `signet/v1`, `admin/v1`, and `signet_client`.
- The `publish` job holds `id-token: write` and no PyPI token secret is referenced anywhere in the file.
- The release notes step produces the annotation and the commits since the previous tag; checked against a throwaway tag in a scratch clone, then deleted.
- `flai check --strict` is clean.

## Notes
