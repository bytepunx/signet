---
id: T-0067
type: task
nature: improvement
title: "README: the client-python/v tag form, the release workflow, and the PyPI trusted publisher"
status: done
parent: S-0012
owner: alex
created: 2026-10-27T22:52:31Z
updated: 2026-11-06T07:51:43Z
transitions:
  - to: ready
    at: 2026-11-06T07:51:38Z
    by: agent-S-0012
  - to: in-progress
    at: 2026-11-06T07:51:38Z
    by: agent-S-0012
  - to: done
    at: 2026-11-06T07:51:43Z
    by: agent-S-0012
stream: S-0012
tags: [client-python]
touches: [client-python/README.md]
after: [T-0064]
usage:
  source: log
  seconds: 5
  estimated: true
  priced_by: estimate
  models:
    - model: claude-sonnet-5-5
      input: 2
      output: 64
      cache_read: 100431
      cache_write: 4599
      cost: 0.0348
---
# T-0067 README: the client-python/v tag form, the release workflow, and the PyPI trusted publisher

## Work

Rewrite the paragraph under `## Install` in `client-python/README.md` that links `../.github/workflows/publish-python.yml` and says release-please tags a `python-v*` release. It waits for T-0064 so the text names a workflow that exists. Replace it with a short `### Releases` subsection that says:

- A release is the tag `client-python/vX.Y.Z`, created by `flai release --pending`; the version is the tag with `client-python/v` removed, written into `pyproject.toml` at build time by `scripts/set-client-python-version.sh`. The checked-in `version` is a placeholder and no commit bumps it.
- [`release-client-python.yml`](../.github/workflows/release-client-python.yml) runs on that tag only: it runs the tests, builds the sdist and wheel, publishes to [PyPI](https://pypi.org/project/signet-client/), and creates the GitHub Release `client-python/vX.Y.Z` with the tag annotation and the commits since the previous tag as its notes. `CHANGELOG.md` is not maintained any more; the releases are the changelog.
- Publishing uses [Trusted Publishing](https://docs.pypi.org/trusted-publishers/) (OIDC, no stored token). The publisher registered on pypi.org for `signet-client` names repository `bytepunx/signet`, workflow `release-client-python.yml`, and no environment; `docs/operators/release.md` has the steps.

Keep the links relative, fix any other mention of `signet-clients`, `python-v*`, or release-please in the README, and leave the sections S-0020 edits (`## Regenerating stubs`) alone.

## Done when

- The README names `client-python/vX.Y.Z`, `release-client-python.yml`, and the trusted publisher's repository, workflow, and empty environment.
- `grep -n -E 'signet-clients|python-v|release-please|publish-python' client-python/README.md` finds nothing.
- Every relative link in the new text resolves to an existing path.
- The markdown lint and `flai check --strict` are clean.

## Notes
