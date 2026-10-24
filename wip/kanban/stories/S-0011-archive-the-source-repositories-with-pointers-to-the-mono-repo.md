---
id: S-0011
type: story
nature: improvement
title: Archive the source repositories with pointers to the mono-repo
status: backlog
parent: E-0001
owner: alex
created: 2026-10-24T22:33:37Z
updated: 2026-10-24T22:33:37Z
transitions: []
tags: []
after: [S-0006, S-0007, S-0009, S-0010]
---
# S-0011 Archive the source repositories with pointers to the mono-repo

## Goal

`bytepunx/signet-proto`, `bytepunx/signet-clients`, and `bytepunx/signet-admin` are archived on GitHub, each with a final commit whose README points to this repository and the folder that replaced it, and their open issues are carried here as stories or closed, so there is one home for signet.

## Acceptance criteria
- [ ] Every open issue and pull request in the three repositories is listed in the story notes with its disposition: a story ID here, closed as done, or closed as won't do.
- [ ] Each repository's README opens with a notice naming this repository and the replacement path (`proto/`, `client-<lang>/`, `admin/`), and links to the last release made from the old repository.
- [ ] The Buf Schema Registry module, Go module proxy entries, and package registry pages for the old names say where the new ones are, as far as each registry allows.
- [ ] The three repositories are archived on GitHub, with the operator's confirmation recorded in the narrative before each archive.
- [ ] `legacy/*` tags in this repository are kept or removed as S-0001 decided, and the decision is written in `docs/contributors/`.
- [ ] The root README no longer calls anything "newly imported"; `design/system/overview.md` links each part to its folder.

## Tasks

## Notes

Archiving and final commits to other repositories reach outside this one: each needs a confirmation per `design/conventions/safety.md`. Operator-only steps go in `## Open questions` of the narrative with the exact `gh` commands.
