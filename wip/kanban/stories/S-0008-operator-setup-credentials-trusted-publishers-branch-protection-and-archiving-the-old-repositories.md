---
id: S-0008
type: story
nature: improvement
title: "Operator setup: credentials, trusted publishers, branch protection, and archiving the old repositories"
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:49Z
updated: 2026-10-27T19:47:50Z
transitions: []
tags: []
topics: [release, github]
touches: [docs/operators/release.md]
after: [S-0005]
---
# S-0008 Operator setup: credentials, trusted publishers, branch protection, and archiving the old repositories

## Goal

The `bytepunx/signet` repository can publish every component: the Actions secrets the release workflows need exist (`BUF_TOKEN`, `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`, `CARGO_REGISTRY_TOKEN`, `NUGET_USER`, `HEX_API_KEY`, and whatever the client stories settle on), PyPI, NuGet, and npm trusted publishing are registered against this repository and the new workflow filenames, `main` is protected with the CI jobs as required checks, and `signet-clients`, `signet-proto`, and `signet-admin` are archived on GitHub with a README pointing here.

## Acceptance criteria
- [ ] `docs/operators/release.md` holds the checklist of secrets and trusted publishers per component, each with where it is registered and which workflow file it names
- [ ] Every item on that checklist is done by the operator and ticked in the story notes with the date
- [ ] Branch protection on `main` requires the CI jobs and the system-flow check
- [ ] The three old repositories are archived and their READMEs point to this repository

## Tasks

## Notes

The agent writes the checklist and the commands (`gh secret set`, `gh api .../branches/main/protection`, `gh repo archive`); creating credentials and archiving repositories is the operator's (safety.md).
