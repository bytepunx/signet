---
id: S-0012
type: story
nature: feature
title: "Python client: release workflow on client-python/v* tags with PyPI trusted publishing"
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-10-27T19:48:21Z
transitions: []
tags: [client-python]
topics: [release, github]
touches: [".github/workflows/release-client-python.yml", client-python/pyproject.toml, client-python/README.md]
after: [S-0007]
---
# S-0012 Python client: release workflow on client-python/v* tags with PyPI trusted publishing

## Goal

A push of `client-python/vX.Y.Z` builds `signet-client` with version `X.Y.Z` taken from the tag (no committed version bump), runs the tests, publishes to PyPI with trusted publishing registered for this repository and `release-client-python.yml`, and creates the GitHub Release. Package metadata names this repository.

## Acceptance criteria
- [ ] `pyproject.toml` takes its version from the tag at build time (for example `hatch-vcs` with the `client-python/v` prefix, or an explicit `--version` step) and names `bytepunx/signet` as the repository
- [ ] `release-client-python.yml` triggers on `client-python/v*` only, tests, builds sdist and wheel, publishes with OIDC, and creates the GitHub Release with notes from the tag annotation
- [ ] `client-python/README.md` names the tag form, the workflow, and the trusted publisher
- [ ] The trusted publisher on pypi.org names this repository and workflow (operator, tracked in the setup story)

## Tasks

## Notes

Reference: signet-clients' `publish-python.yml` on GitHub.
