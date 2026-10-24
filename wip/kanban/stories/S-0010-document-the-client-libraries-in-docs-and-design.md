---
id: S-0010
type: story
nature: improvement
title: Document the client libraries in docs/ and design/
status: ready
parent: E-0001
owner: alex
created: 2026-10-24T22:33:37Z
updated: 2026-10-24T22:33:48Z
transitions:
  - to: ready
    at: 2026-10-24T22:33:48Z
    by: claude
tags: []
---
# S-0010 Document the client libraries in docs/ and design/

## Goal

Each client library's behaviour (workload connection, watch, restart lock, SOPS encrypt, admin connection) is described once in `design/system/` as the client contract, and `docs/users/` has one page per language that says how to install, connect, and use it, so the per-client READMEs shrink to a pointer plus language specifics.

## Acceptance criteria
- [ ] `design/system/` has a client contract page describing the behaviour every client implements and the differences per language, with `topics` naming the client projects.
- [ ] `docs/users/` has a page for Go, Python, TypeScript, Rust, and C#, each covering install from its registry, connecting to signetd, reading and watching a secret, the restart lock, and SOPS encryption, with links to the examples under `client-<lang>/examples/`.
- [ ] `docs/users/` states the status of the Erlang client and the admin UI as they are (not started), with the constraints from `client-erlang/README.md` captured in `design/system/` or a backlog story.
- [ ] Each `client-*/README.md` links to its `docs/users/` page and keeps only what is specific to building or developing that client.
- [ ] `make lint-md` passes and every link resolves.

## Tasks

## Notes

The READMEs are the source today; check them against the code before copying claims.
