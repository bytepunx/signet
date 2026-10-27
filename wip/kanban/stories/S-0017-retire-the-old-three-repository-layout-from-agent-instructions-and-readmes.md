---
id: S-0017
type: story
nature: remediation
title: Retire the old three-repository layout from agent instructions and READMEs
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-10-27T19:48:20Z
transitions: []
tags: [server]
topics: [docs]
touches: [server/CLAUDE.md, admin/CLAUDE.md, server/README.md, admin/README.md, docs/contributors/index.md, design/system/overview.md, server/githooks/pre-commit]
---
# S-0017 Retire the old three-repository layout from agent instructions and READMEs

## Goal

No file in the repository describes signet, signet-proto, signet-clients, and signet-admin as separate repositories. `server/CLAUDE.md` and `admin/CLAUDE.md`, which predate the root `CLAUDE.md` and contradict it, are deleted after anything still true moves into the root conventions' project additions or `docs/contributors/`. `server/README.md`'s protocol section, `admin/README.md`'s related-repos section, and `design/system/overview.md`'s placeholder diagram describe the mono-repo. The server pre-commit hook runs from `server/`.

## Acceptance criteria
- [ ] `grep -r "signet-clients\|signet-proto\|signet-admin"` finds only the BSR module name `buf.build/bytepunx/signet-proto` and historical changelog entries
- [ ] `server/CLAUDE.md` and `admin/CLAUDE.md` are gone and `docs/contributors/index.md` tells a contributor how each component is built and tested
- [ ] `design/system/overview.md` shows the real parts (server, proto, admin, the clients) and how they relate

## Tasks

## Notes
