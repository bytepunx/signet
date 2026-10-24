---
id: S-0007
type: story
nature: improvement
title: "Retire stale cross-repo references and sub-project agent files; add the root LICENSE"
status: backlog
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-10-24T22:33:36Z
transitions: []
tags: []
after: [S-0003]
---
# S-0007 Retire stale cross-repo references and sub-project agent files; add the root LICENSE

## Goal

Nothing in the tree describes `signet-proto`, `signet-clients`, or `signet-admin` as where code lives; package metadata, READMEs, Dockerfiles, Helm files, and agent instructions point at this repository and its paths; and the repository carries the licence S-0001 decided.

## Acceptance criteria
- [ ] `grep -rI 'signet-proto\|signet-clients\|signet-admin'` over the tree (excluding `.git`, generated code, and dependency folders) matches only `CHANGELOG.md` history, legacy tag names, and sentences that deliberately describe the migration.
- [ ] `server/CLAUDE.md` and `admin/CLAUDE.md` are removed; anything in them still true and not in the root `CLAUDE.md` or `design/` moves there as project additions or living design.
- [ ] `server/CHECKLIST.md` is removed, with any unfinished item turned into a backlog story.
- [ ] Package metadata names this repository: `client-python/pyproject.toml` URLs, `client-typescript/package.json` `repository`, `client-rust/Cargo.toml` `repository`, `client-csharp/SignetClient.csproj` package URLs, `server/deploy/helm/signet/Chart.yaml` sources.
- [ ] Each example Dockerfile under `client-*/examples/` builds from this repository's layout.
- [ ] Every `README.md` under `server/`, `proto/`, `admin/`, and `client-*/` opens by saying it is a part of this repository and links to the root README; the root README lists every sub-project with one line each.
- [ ] A root `LICENSE` exists per the ADR, and each sub-project either has no copy or an identical one.
- [ ] Test names and comments that cite old issue numbers (`bytepunx/signet-clients#64`, `#67`) say what the issue was instead of linking to it, or link to the archived issue URL.

## Tasks

## Notes

Survey count on 2026-10-24: 74 mentions of `signet-proto`, 203 of `signet-clients`, 56 of `signet-admin`, across about 90 files.
