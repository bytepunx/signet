---
id: T-0026
type: task
nature: improvement
title: Remove client-erlang as a component and keep its toolchain note in the Elixir client
status: backlog
parent: S-0006
owner: alex
created: 2026-10-27T22:25:22Z
updated: 2026-10-27T22:25:22Z
transitions: []
stream: S-0006
tags: [client-elixir, client-erlang]
touches: [client-erlang/README.md, client-elixir/README.md, client-elixir/buf.gen.yaml, system-flow.yaml]
---
# T-0026 Remove client-erlang as a component and keep its toolchain note in the Elixir client

## Work

Fold the substance of `client-erlang/README.md` into `client-elixir/README.md`, in or beside "Regenerating stubs": Buf's registry hosts no remote plugin for Erlang or Elixir, which is why `buf.gen.yaml` names `protoc-gen-elixir` as a local plugin; and an Erlang client would take the same shape, with `gpb`'s `protoc-gen-gpb` for messages and either `grpcbox` or the Elixir client's `grpc` package for the service layer. Replace the link to `../client-erlang/README.md` with that text.

Update the comment in `client-elixir/buf.gen.yaml` that points at `../client-erlang/README.md` so it points at the Elixir README section instead.

Delete `client-erlang/README.md` with `git rm`, so the folder is gone, and remove the component with `flai subproject remove client-erlang` (the manifest is flai's to change, not a hand edit). Search the repository for any other `client-erlang` reference outside `wip/` and fix it.

This is one task because the note must exist in its new home in the same commit that deletes its old one.

## Done when

- `client-erlang/` does not exist and `system-flow.yaml` lists no `client-erlang` project.
- `client-elixir/README.md` keeps the note on why Buf's registry cannot generate BEAM stubs and what an Erlang client would use.
- No file outside `wip/` links to `client-erlang`.
- `flai release --pending --dry-run` plans nothing for `client-erlang`.
- `flai check --strict` is clean.

## Notes

Layer 1. It shares no path with the other tasks of S-0006, so it runs together with them.
