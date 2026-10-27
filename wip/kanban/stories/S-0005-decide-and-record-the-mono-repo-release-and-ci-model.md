---
id: S-0005
type: story
nature: research
title: Decide and record the mono-repo release and CI model
status: review
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:49Z
updated: 2026-10-27T21:28:10Z
transitions:
  - to: ready
    at: 2026-10-27T19:47:51Z
    by: arobson
  - to: in-progress
    at: 2026-10-27T21:21:54Z
    by: agent-S-0005
  - to: review
    at: 2026-10-27T21:28:10Z
    by: agent-S-0005
tags: []
topics: [release, ci, github]
touches: [design/adrs/README.md, design/system/release.md, design/system/README.md, docs/operators/release.md, docs/operators/index.md, design/conventions/git.md, design/adrs/0001-the-mono-repo-releases-from-flai-tags-per-component-with-github-actions.md]
agent:
  harness: claude-code
usage:
  source: log
  seconds: 391
  priced_by: harness
  turns:
    - day: 2026-10-27
      ceremony: 1
      test_runs: 1
      hand_edits: 1
      work: 21
  models:
    - model: claude-fable-5-1
      input: 770
      output: 20698
      cache_read: 2226396
      cache_write: 123555
      cost: 4.0703
---
# S-0005 Decide and record the mono-repo release and CI model

## Goal

One accepted ADR records how the mono-repo releases, and the living design and operator runbook describe it, so every workflow story under E-0002 builds the same model. The epic's notes carry the recommended model; this story confirms or amends it with the operator and writes it down.

## Acceptance criteria
- [x] An ADR under `design/adrs/` (written with `flai adr new`) is accepted by the operator and decides: `flai release --pending` is the only version authority; tags are `<component>/vX.Y.Z` with the component names of `system-flow.yaml`; baseline tags are placed on the import commits so numbering continues from the old repositories; package versions are derived from the tag at build time and no version file is bumped in a commit; the changelog of a component release is its GitHub Release, built from the tag annotation and the commits since the previous tag of that component; in-repo consumers generate stubs from `proto/proto` by directory input while the BSR module stays for external consumers; the old repositories are archived once publishing works here
- [x] `design/system/release.md` describes the flow from acceptance to published artifact per component, naming each workflow file, trigger, artifact, registry, and credential, with a Mermaid diagram, and `design/system/README.md` lists it
- [x] `docs/operators/release.md` is a runbook: how to publish (`git fetch`, `flai release --pending`), what each component's release produces, where to find it, and the credentials and trusted publishers the workflows need, with `docs/operators/index.md` linking it
- [x] `design/conventions/git.md` project additions name the tag scheme and the workflow naming (`ci.yml`, `release-<component>.yml`)

## Tasks
- T-0006 ADR: mono-repo release and CI model
- T-0007 Living design: design/system/release.md
- T-0008 Operator runbook: docs/operators/release.md
- T-0009 git.md project additions: tag scheme and workflow naming

## Notes

A research story: its findings land on main and contribute to no component's release.
