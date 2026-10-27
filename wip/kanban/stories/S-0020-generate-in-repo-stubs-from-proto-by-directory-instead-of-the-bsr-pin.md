---
id: S-0020
type: story
nature: improvement
title: Generate in-repo stubs from proto/ by directory instead of the BSR pin
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:48:36Z
updated: 2026-10-27T19:48:36Z
transitions: []
tags: [server, proto]
topics: [proto]
touches: [server/buf.gen.yaml, client-go/buf.gen.yaml, client-python/buf.gen.yaml, client-typescript/buf.gen.yaml, client-rust/buf.gen.yaml, client-csharp/buf.gen.yaml, client-elixir/buf.gen.yaml]
---
# S-0020 Generate in-repo stubs from proto/ by directory instead of the BSR pin

## Goal

Every in-repo consumer of the schema (the server and the six clients) generates its stubs from `../proto/proto` as a directory input in its `buf.gen.yaml`, so a schema change and the code that uses it land in one pull request and the `proto-bump.yml` automation of the old repository is not needed. The BSR pin stays only in documentation for external consumers.

## Acceptance criteria
- [ ] Each `buf.gen.yaml` names `directory: ../proto/proto` as its input and `buf generate` produces stubs identical to those from the `v0.4.0` BSR pin at the same schema
- [ ] `make test` passes for every component
- [ ] Each component's README no longer tells readers to bump a BSR pin

## Tasks

## Notes

Touches every component, so publishing it releases a patch of each; that is correct, since each ships regenerated stubs. Tagged `server` and `proto`: a story touching several components must name the one it delivers to; the others get a patch. Depends on the ADR deciding directory input.
