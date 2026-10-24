---
id: S-0002
type: story
nature: improvement
title: Generate every consumer's stubs from the in-repo proto module
status: backlog
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-10-24T22:33:36Z
transitions: []
tags: []
after: [S-0001]
---
# S-0002 Generate every consumer's stubs from the in-repo proto module

## Goal

Server and all five generated clients take their schema from `proto/` as S-0001 decided, so a proto change and its consumers' regeneration land in one commit, and nothing in the tree reads the schema from the Buf Schema Registry unless the ADR keeps the registry as a published mirror.

## Acceptance criteria
- [ ] `server/buf.gen.yaml` and `client-{csharp,go,python,rust,typescript}/buf.gen.yaml` take their input from `proto/` (buf workspace or `directory` input), with plugin pins unchanged.
- [ ] `buf generate` in each of those six folders succeeds from a clean clone and produces no diff against the committed generated code where generated code is committed.
- [ ] `proto/buf.yaml` lint and breaking checks still pass, with breaking compared against the last `proto/v*` tag.
- [ ] Generated-code policy is the same in every project as the ADR states, and each `.gitignore` matches it.
- [ ] Comments in every `buf.gen.yaml` and the proto, server, and client READMEs no longer describe the schema as living in another repository.
- [ ] `make proto` at the repository root, backed by a script in `scripts/`, regenerates every consumer.

## Tasks

## Notes

Today every generator pins `buf.build/bytepunx/signet-proto:v0.4.0`. `server/.github/workflows/ci.yml` has a proto-check job that auto-commits regenerated stubs; S-0005 decides what replaces it.
