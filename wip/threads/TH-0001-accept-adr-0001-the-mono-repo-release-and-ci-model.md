---
id: TH-0001
title: Accept ADR-0001, the mono-repo release and CI model
anchor:
  path: wip/kanban/stories/S-0005-decide-and-record-the-mono-repo-release-and-ci-model.md
  item: S-0005
status: resolved
participants: [agent-S-0005, alex]
created: 2026-10-27T21:24:44Z
updated: 2026-10-27T21:27:31Z
---

# TH-0001 Accept ADR-0001, the mono-repo release and CI model

On wip/kanban/stories/S-0005-decide-and-record-the-mono-repo-release-and-ci-model.md.

## Entries

### 2026-10-27T21:24:44Z agent-S-0005
ADR-0001 is committed on story/S-0005 as proposed: design/adrs/0001-the-mono-repo-releases-from-flai-tags-per-component-with-github-actions.md. It records the model from E-0002's notes and S-0005's first criterion unchanged: flai release --pending as the only version authority; <component>/vX.Y.Z tags with system-flow.yaml names; baseline tags on the import commits (server/v0.11.0, proto/v0.4.0, client-go/v0.7.0, client-python/v0.5.0, client-rust/v0.6.0, client-typescript/v0.9.0, client-csharp/v0.6.0); versions derived from the tag at build time with no version-file bumps; the GitHub Release as the changelog; directory-input stubs for in-repo consumers with the BSR kept for external ones; old repositories archived once publishing works here.

Recommended answer: accept as written. On "accept" I set its status to accepted and tick criterion 1. On an amendment, I rewrite the proposed ADR (it is not yet accepted, so editing it is allowed) and the design and runbook that follow from it. The rest of the story (design/system/release.md, docs/operators/release.md, git.md additions) proceeds meanwhile against the ADR as proposed.

### 2026-10-27T21:26:08Z alex
accept as written

### 2026-10-27T21:27:31Z agent-S-0005
Resolved: Operator accepted ADR-0001 as written; status set to accepted in commit 9918473 on story/S-0005.
