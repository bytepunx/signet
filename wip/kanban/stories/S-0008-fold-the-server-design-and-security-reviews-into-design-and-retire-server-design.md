---
id: S-0008
type: story
nature: improvement
title: Fold the server design and security reviews into design/ and retire server/design
status: ready
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-10-24T22:33:48Z
transitions:
  - to: ready
    at: 2026-10-24T22:33:48Z
    by: claude
tags: []
---
# S-0008 Fold the server design and security reviews into design/ and retire server/design

## Goal

The living design of signet is in `design/system/`, split by topic and kept current, and the security reviews are recorded where `design/` expects them, so `server/design/` is empty and removed and agents find the design where the conventions say it is.

## Acceptance criteria
- [ ] `server/design/draft.md` is split into topic files under `design/system/` (at least: storage, encryption and unsealing, identity and policy, API surface, GitOps and SOPS, audit, admin access, deployment), each with front matter and `topics`, and `design/system/README.md` indexes them.
- [ ] Each decision in `draft.md` that is architectural has an accepted ADR under `design/adrs/` dated to when it was taken, with the living design linking to it.
- [ ] `server/design/security-review-2026-08-31.md` and `security-findings.md` are moved under `design/` (an issues or security section), with every open finding recorded in `design/issues/` and `summary.md` current.
- [ ] `server/design/` no longer exists; nothing links to it.
- [ ] `flai check --strict` passes with no `doc.topics` warnings.

## Tasks

## Notes

Keep history: `git mv` where a file moves whole. The design sections should describe what is, not the draft's deliberation; deliberation that still matters goes into the ADR's context.
