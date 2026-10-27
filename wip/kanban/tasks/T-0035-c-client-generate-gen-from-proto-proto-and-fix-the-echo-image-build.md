---
id: T-0035
type: task
nature: improvement
title: "C# client: generate gen/ from ../proto/proto and fix the echo image build"
status: backlog
parent: S-0020
owner: alex
created: 2026-10-27T22:25:44Z
updated: 2026-10-27T22:25:44Z
transitions: []
stream: S-0020
tags: [client-csharp, proto]
touches: [client-csharp/buf.gen.yaml, client-csharp/README.md, client-csharp/examples/echo/Dockerfile, client-csharp/examples/echo/Dockerfile.dockerignore]
---
# T-0035 C# client: generate gen/ from ../proto/proto and fix the echo image build

## Work

Replace the pinned BSR module input in `client-csharp/buf.gen.yaml` with `directory: ../proto/proto` and rewrite its comment. The echo image's `proto` stage runs `buf generate` from a copy of `buf.gen.yaml` alone, which a relative directory input breaks: change the Dockerfile to build from the repository root, copy `proto/proto` and `client-csharp/buf.gen.yaml` to the same relative layout the repository has, and run `buf generate` there; update `Dockerfile.dockerignore` and the build comment to match. In `client-csharp/README.md` rewrite the opening paragraph and the regenerate section (about lines 4 and 26 to 32) so they say `buf generate` reads `../proto/proto` and no longer tell readers to bump a BSR pin.

## Done when

- `client-csharp/buf.gen.yaml` names `directory: ../proto/proto` as its only input.
- `buf generate` in `client-csharp/` writes the same `gen/` files as the `v0.4.0` pin did.
- `dotnet test` in `client-csharp/` passes.
- The echo image builds from the repository root with the documented command.
- `client-csharp/README.md` no longer tells readers to bump a BSR pin.

## Notes
