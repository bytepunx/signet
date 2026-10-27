---
id: T-0006
type: task
nature: research
title: "ADR: mono-repo release and CI model"
status: done
parent: S-0005
owner: arobson
created: 2026-10-27T21:23:45Z
updated: 2026-10-27T21:24:44Z
transitions:
  - to: ready
    at: 2026-10-27T21:23:45Z
    by: agent-S-0005
  - to: in-progress
    at: 2026-10-27T21:23:45Z
    by: agent-S-0005
  - to: done
    at: 2026-10-27T21:24:44Z
    by: agent-S-0005
stream: S-0005
tags: []
touches: [design/adrs/0001-mono-repo-release-and-ci-model.md, design/adrs/README.md]
usage:
  source: log
  seconds: 59
  estimated: true
  priced_by: harness
  models:
    - model: claude-fable-5-1
      input: 91
      output: 2457
      cache_read: 264334
      cache_write: 14669
      cost: 0.4833
---
# T-0006 ADR: mono-repo release and CI model

## Work

Write the ADR with `flai adr new --commit` from the recommended model in E-0002's notes and S-0005's first acceptance criterion: flai as the only version authority, `<component>/vX.Y.Z` tags, baseline tags on the import commits, versions derived from the tag at build time, GitHub Release as the changelog, directory-input stubs for in-repo consumers with the BSR for external ones, old repositories archived once publishing works here. Open a thread asking the operator to accept it.

## Done when

The ADR file and its row in `design/adrs/README.md` are committed on `story/S-0005` and a thread on S-0005 asks the operator to accept it.

## Notes
