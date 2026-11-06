---
id: S-0002
type: story
nature: improvement
title: Generate every consumer's stubs from the in-repo proto module
status: backlog
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-11-03T18:25:45Z
transitions: []
tags: []
after: [S-0001]
---
# S-0002 Generate every consumer's stubs from the in-repo proto module

## Goal

Server and all five generated clients take their schema from `proto/` through a buf `directory: ../proto` input, as [ADR-0001](../../../design/adrs/0001-every-consumer-generates-its-stubs-from-proto-through-a-buf-directory-input-the.md) decides, so a proto change and its consumers' regeneration land in one commit. Nothing in the tree reads the schema from the Buf Schema Registry: `buf.build/bytepunx/signet-proto` stays only as a mirror pushed from `proto/` for consumers outside the repository. Generated stubs are committed in `server/gen` and `client-go/gen` only; C#, Python, Rust, and TypeScript ignore theirs and run `buf generate` before they build.

## Acceptance criteria
- [ ] `server/buf.gen.yaml` and `client-{csharp,go,python,rust,typescript}/buf.gen.yaml` take their input from `proto/` with `inputs: - directory: ../proto`, with plugin pins unchanged.
- [ ] `buf generate` in each of those six folders succeeds from a clean clone and produces no diff against the committed generated code in `server/gen` and `client-go/gen`.
- [ ] `proto/buf.yaml` lint and breaking checks still pass, with breaking compared against the last `proto/v*` tag.
- [ ] Generated code is committed in `server/gen` and `client-go/gen` and ignored everywhere else, as ADR-0001 states: `client-typescript/src/gen` is removed from the index and added to its `.gitignore`, and each `.gitignore` matches the policy.
- [ ] The Python wheel and every other client package still ship their generated stubs when built from a clean clone (hatchling needs ignored stubs listed as `artifacts`).
- [ ] Comments in every `buf.gen.yaml` and the proto, server, and client READMEs no longer describe the schema as living in another repository or pinned to a registry version.
- [ ] `make proto` at the repository root, backed by a script in `scripts/`, regenerates every consumer.

## Tasks

## Notes

Waits for S-0001: it reads ADR-0001, which S-0001 records, for the input mechanism and the generated-code policy.

Today every generator pins `buf.build/bytepunx/signet-proto:v0.4.0`. `server/.github/workflows/ci.yml` has a proto-check job that auto-commits regenerated stubs; S-0005 decides what replaces it, and ADR-0001 asks for a CI check that `server/gen` and `client-go/gen` match a fresh `buf generate`. The BSR push and label workflows move to the root in S-0006, per ADR-0001 and ADR-0003.
