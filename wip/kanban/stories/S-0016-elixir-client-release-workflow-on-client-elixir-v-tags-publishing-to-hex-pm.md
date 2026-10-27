---
id: S-0016
type: story
nature: feature
title: "Elixir client: release workflow on client-elixir/v* tags publishing to hex.pm"
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-10-27T19:48:21Z
transitions: []
tags: [client-elixir]
topics: [release, github]
touches: [".github/workflows/release-client-elixir.yml", client-elixir/mix.exs, client-elixir/README.md, client-elixir/CHANGELOG.md]
after: [S-0007]
---
# S-0016 Elixir client: release workflow on client-elixir/v* tags publishing to hex.pm

## Goal

A push of `client-elixir/vX.Y.Z` runs `scripts/test-client-elixir.sh`, sets the package version from the tag, publishes `signet_client` to hex.pm with `HEX_API_KEY`, and creates the GitHub Release. `mix.exs` reads its version from an environment variable the workflow sets, defaulting to `0.0.0-dev`, and the hand-written `CHANGELOG.md` is retired in favour of the GitHub Release.

## Acceptance criteria
- [ ] `release-client-elixir.yml` triggers on `client-elixir/v*` only, tests in Docker, publishes with `mix hex.publish --yes`, and creates the GitHub Release with notes from the tag annotation
- [ ] `mix.exs` takes its version from the tag at build time and its package metadata names `bytepunx/signet`
- [ ] `client-elixir/README.md` names the tag form and the workflow, and its install section points at hex.pm

## Tasks

## Notes
