---
id: S-0012
type: story
nature: feature
title: "Python client: release workflow on client-python/v* tags with PyPI trusted publishing"
status: review
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-11-06T07:54:41Z
transitions:
  - to: ready
    at: 2026-11-06T07:50:00Z
    by: alex
  - to: in-progress
    at: 2026-11-06T07:50:10Z
    by: agent-S-0012
  - to: review
    at: 2026-11-06T07:53:48Z
    by: agent-S-0012
tags: [client-python]
topics: [release, github]
touches: [".github/workflows/release-client-python.yml", client-python/pyproject.toml, client-python/README.md, scripts/set-client-python-version.sh]
after: [S-0007]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
usage:
  source: log
  seconds: 229
  priced_by: harness
  turns:
    - day: 2026-11-06
      ceremony: 2
      work: 18
  models:
    - model: claude-sonnet-5-5
      input: 46
      output: 10548
      cache_read: 1763862
      cache_write: 97482
      cost: 0.6719
  strategic:
    - kind: orchestrator
      seconds: 6
      estimated: true
      models:
        - model: claude-sonnet-5-5
          input: 6
          output: 37
          cache_read: 187386
          cache_write: 30260
          cost: 0.1099
forecast:
  duration: 2h
  delivery: 2026-11-14T15:38:00Z
  basis: "Its own forecast of 2h; 3rd in the pull order with an in-progress limit of 2, behind S-0020 and S-0011."
  by: flai
  at: 2026-11-06T07:49:36Z
---
# S-0012 Python client: release workflow on client-python/v* tags with PyPI trusted publishing

## Goal

A push of `client-python/vX.Y.Z` builds `signet-client` with version `X.Y.Z` taken from the tag (no committed version bump), runs the tests, publishes to PyPI with trusted publishing registered for this repository and `release-client-python.yml`, and creates the GitHub Release. Package metadata names this repository.

## Acceptance criteria
- [x] `pyproject.toml` takes its version from the tag at build time (for example `hatch-vcs` with the `client-python/v` prefix, or an explicit `--version` step) and names `bytepunx/signet` as the repository
- [x] `release-client-python.yml` triggers on `client-python/v*` only, tests, builds sdist and wheel, publishes with OIDC, and creates the GitHub Release with notes from the tag annotation
- [x] `client-python/README.md` names the tag form, the workflow, and the trusted publisher
- [x] The trusted publisher on pypi.org names this repository and workflow (operator, tracked in the setup story)

## Tasks
- T-0059 pyproject.toml: placeholder version and project URLs naming bytepunx/signet
- T-0060 Script that writes the tag's version into pyproject.toml
- T-0064 Write release-client-python.yml: test, build, publish to PyPI, and GitHub Release on client-python/v* tags
- T-0067 README: the client-python/v tag form, the release workflow, and the PyPI trusted publisher

## Notes

Reference: signet-clients' `publish-python.yml` on GitHub.

### Planning

Touches, by source:

- Declared: `.github/workflows/release-client-python.yml`, `client-python/pyproject.toml`, `client-python/README.md`. All kept.
- Design: `design/system/release.md` ("Versions in packages") says the version lands in `pyproject.toml` `version` at build time, which needs a step the workflow can call. `scripts/set-client-python-version.sh` is added (T-0060), as `tooling.md` asks for scripts over inline command sequences.
- Co-change: `flai touches suggest` listed only `client-python/CHANGELOG.md`, `src/signet_client/__init__.py`, `admin.py`, and `tests/test_admin.py` (2 to 5 of 13 commits). None is taken: ADR-0001 retires `CHANGELOG.md`, and the Python sources are untouched by a release workflow.
- Layout: no folder touch is kept.

Decisions:

- Version: an explicit rewrite of the `version` line, not `hatch-vcs`. The repository holds eight components with tags `<component>/v...`, and hatch-vcs would need a custom tag match to ignore all but this prefix and a full-history checkout in every job. A one-line rewrite is checked in one script run. The criterion allows either.
- Pre-release tags: the script accepts only `X.Y.Z` with an optional PEP 440 `a`, `b`, or `rc` suffix, because PyPI rejects anything else after the build.
- No test publish: criterion 2 requires the workflow to trigger on `client-python/v*` only, so there is no `workflow_dispatch` dry run, and any matching tag publishes to the real PyPI. The tasks verify the build and the release notes locally. The first live run is the first release.

After: S-0007 only (already declared). T-0064 calls `scripts/test-client-python.sh`, which S-0007 creates. S-0008 (the PyPI registration, criterion 4) is not an `after`: it is needed only for a live run, not for the change to merge.

Overlaps to know of, not `after` reasons: S-0020 edits `client-python/README.md` (the `## Regenerating stubs` section) and `client-python/buf.gen.yaml`; T-0067 edits the `## Install` section, so the two do not share lines. S-0006 edits `docs/operators/release.md`, which this story only reads.

Layers of tasks:

| Layer | Tasks |
|-------|-------|
| 1 | T-0059 pyproject.toml, T-0060 version script |
| 2 | T-0064 release-client-python.yml |
| 3 | T-0067 README |

Forecast: flai gave 14m (size 7: 4 criteria, 3 touches, a median of 116 s per unit over three done stories, all Elixir client work). Adjusted to 2h: the workflow has four jobs and the wheel check needs a local Docker build with `buf generate` before it can be trusted; the other three tasks are small. Delivery moves to 2026-10-28T12:02:00Z with it.

Cost of delay: not set. `flai cod` stops because neither this story nor E-0002 has the operator's inputs; asked on the plan thread.
