---
id: S-0016
type: story
nature: feature
title: "Elixir client: release workflow on client-elixir/v* tags publishing to hex.pm"
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-11-06T07:20:12Z
transitions: []
tags: [client-elixir]
topics: [release, github]
touches: [".github/workflows/release-client-elixir.yml", client-elixir/mix.exs, client-elixir/README.md, client-elixir/CHANGELOG.md, scripts/publish-client-elixir.sh, scripts/README.md, design/system/release.md]
after: [S-0007]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
forecast:
  duration: 2h
  delivery: 2026-11-16T20:39:00Z
  basis: "Its own forecast of 2h; 8th in the pull order with an in-progress limit of 2, behind S-0029, S-0031, S-0032, S-0020, S-0011, S-0012, S-0013, S-0014 and S-0015."
  by: flai
  at: 2026-11-06T07:20:12Z
---
# S-0016 Elixir client: release workflow on client-elixir/v* tags publishing to hex.pm

## Goal

A push of `client-elixir/vX.Y.Z` runs `scripts/test-client-elixir.sh`, sets the package version from the tag, publishes `signet_client` to hex.pm with `HEX_API_KEY`, and creates the GitHub Release. `mix.exs` reads its version from an environment variable the workflow sets, defaulting to `0.0.0-dev`, and the hand-written `CHANGELOG.md` is retired in favour of the GitHub Release.

## Acceptance criteria
- [ ] `release-client-elixir.yml` triggers on `client-elixir/v*` only, tests in Docker, publishes with `mix hex.publish --yes`, and creates the GitHub Release with notes from the tag annotation
- [ ] `mix.exs` takes its version from the tag at build time and its package metadata names `bytepunx/signet`
- [ ] `client-elixir/README.md` names the tag form and the workflow, and its install section points at hex.pm

## Tasks
- T-0068 mix.exs reads its version from SIGNET_CLIENT_VERSION and names bytepunx/signet
- T-0075 scripts/publish-client-elixir.sh publishes signet_client to hex.pm from Docker
- T-0082 Write release-client-elixir.yml: test, publish to hex.pm, and create the GitHub Release on client-elixir/v* tags
- T-0084 Retire CHANGELOG.md and document the client-elixir/v tag form, workflow, and hex.pm install

## Notes

### Planning

Touches, by source:

- Declared: `.github/workflows/release-client-elixir.yml`, `client-elixir/mix.exs`, `client-elixir/README.md`, `client-elixir/CHANGELOG.md`. All kept.
- Layout: `scripts/publish-client-elixir.sh` and `scripts/README.md`. The tooling convention puts command sequences in `scripts/`, and `scripts/test-client-elixir.sh` passes no environment into its container, so a publish step needs a script of its own that passes `SIGNET_CLIENT_VERSION` and `HEX_API_KEY`. `scripts/README.md` lists every script.
- Design: `design/system/release.md`. Its "Versions in packages" row for `client-elixir` says `mix.exs` `@version`, which the environment variable changes.
- Co-change: `flai touches suggest` listed `docs/users/elixir-client.md` (3 of 5 commits), `client-elixir/lib/signet.ex`, and `design/tech/README.md`. None was taken: the user page has no install or changelog text to change, and the other two follow code and dependency changes, not releases.

No folder touch is kept.

Overlaps to know of, not `after` reasons:

- S-0020 edits `client-elixir/README.md` (stub generation section) and `scripts/test-client-elixir.sh`; this story edits the README's install section and adds a `## Releasing` section, and does not touch the test script.
- S-0006 edits `client-elixir/README.md` (its `client-erlang` note) and `client-elixir/buf.gen.yaml`.
- S-0007 and S-0008 edit `scripts/README.md`, each adding rows.

After: `S-0007` as declared, kept, because its root `ci.yml` and the per-component test scripts are what the release workflow's "build and test as CI does" step mirrors. Nothing else it needs is unaccepted: `scripts/test-client-elixir.sh` and `client-elixir/Dockerfile` are on main, and the `HEX_API_KEY` of S-0008 is needed only for a real release, which this story does not run.

Layers of tasks:

| Layer | Task |
|-------|------|
| 1 | T-0068 `mix.exs` |
| 2 | T-0075 publish script, which needs `mix.exs` to read the variable for its dry run |
| 3 | T-0082 workflow, which calls the script |
| 4 | T-0084 README, `CHANGELOG.md`, design row, which name what the earlier tasks built |

Forecast: flai gave 14m (116 s per unit of size over 3 done stories, size 7), which counts neither the Docker image build for the dry run nor the one-off care a workflow needs. Adjusted to 2h for four small tasks, one local Docker dry run, and no way to run the workflow itself before the operator's first release. Delivery is flai's 2026-10-28T11:09:00Z plus the extra 1h46m.

Cost of delay: not set. `flai cod` stopped because neither the story nor E-0002 has the operator's inputs; asked on the plan thread.

No test publish: a Hex release can be retired but not deleted after an hour, so the first real run is the operator's `client-elixir/v0.1.0`, after S-0008 sets `HEX_API_KEY`.
