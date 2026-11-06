---
id: S-0013
type: story
nature: feature
title: "TypeScript client: release workflow on client-typescript/v* tags with npm provenance"
status: done
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-11-06T07:53:46Z
transitions:
  - to: ready
    at: 2026-11-06T07:50:02Z
    by: alex
  - to: in-progress
    at: 2026-11-06T07:50:19Z
    by: agent-S-0013
  - to: review
    at: 2026-11-06T07:53:07Z
    by: agent-S-0013
  - to: done
    at: 2026-11-06T07:53:46Z
    by: alex
tags: [client-typescript]
topics: [release, github]
touches: [".github/workflows/release-client-typescript.yml", client-typescript/package.json, client-typescript/package-lock.json, client-typescript/README.md]
after: [S-0007]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
usage:
  source: log
  seconds: 175
  priced_by: harness
  turns:
    - day: 2026-11-06
      ceremony: 1
      work: 19
  models:
    - model: claude-sonnet-5-5
      input: 44
      output: 5198
      cache_read: 1428209
      cache_write: 65623
      cost: 0.4574
  strategic:
    - kind: orchestrator
      seconds: 63
      estimated: true
      models:
        - model: claude-sonnet-5-5
          input: 29
          output: 181
          cache_read: 819909
          cache_write: 110168
          cost: 0.4727
---
# S-0013 TypeScript client: release workflow on client-typescript/v* tags with npm provenance

## Goal

A push of `client-typescript/vX.Y.Z` sets the package version from the tag, builds, tests, publishes `@bytepunx/signet-client` to npm with provenance from this repository, and creates the GitHub Release. `package.json` names this repository.

## Acceptance criteria
- [x] `release-client-typescript.yml` triggers on `client-typescript/v*` only, sets the version with `npm version --no-git-tag-version`, builds, tests, publishes with `--provenance`, and creates the GitHub Release with notes from the tag annotation
- [x] `package.json` repository and bugs fields name `bytepunx/signet`
- [x] `client-typescript/README.md` names the tag form and the workflow

## Tasks
- T-0097 Release workflow, package metadata, README

## Notes

S-0013 waits for S-0007 because it builds on the root `ci.yml` and the shared release-workflow shape S-0007 writes: the package build and test step it reuses, and the check names it adds to the required checks.

Reference: signet-clients' `publish-typescript.yml` on GitHub, which used OIDC rather than a stored token.
