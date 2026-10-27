---
id: S-0018
type: story
nature: research
title: Decide the admin sub-project's place in the mono-repo
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-10-27T19:47:50Z
transitions: []
tags: [admin]
topics: [admin]
touches: [admin/README.md, system-flow.yaml]
---
# S-0018 Decide the admin sub-project's place in the mono-repo

## Goal

`admin/` holds a README and nothing else but is registered as a SvelteKit project, so `flai check` warns about missing examples and a release could tag an empty component. The operator decides whether to scaffold the SvelteKit app now (as a story under its own epic) or remove `admin` from `system-flow.yaml` and keep the README as a design note under `design/system/` until the work starts.

## Acceptance criteria
- [ ] The decision is recorded in `design/system/overview.md` and, if the project is removed, `system-flow.yaml` and the folder reflect it
- [ ] `flai check --strict` reports nothing about `admin`

## Tasks

## Notes
