---
id: T-0005
type: task
nature: research
title: Write design/tech/ entries for every technology the sub-projects depend on
status: done
parent: S-0001
owner: alex
created: 2026-10-24T22:53:06Z
updated: 2026-11-03T18:21:23Z
transitions:
  - to: ready
    at: 2026-11-03T18:17:26Z
    by: agent-S-0001
  - to: in-progress
    at: 2026-11-03T18:17:26Z
    by: agent-S-0001
  - to: done
    at: 2026-11-03T18:21:23Z
    by: agent-S-0001
stream: S-0001
tags: [design, tech]
touches: [design/tech/README.md, design/tech/go.md, design/tech/buf.md, design/tech/grpc-protobuf.md, design/tech/cockroachdb.md, design/tech/spire.md, design/tech/helm.md, design/tech/sops.md, design/tech/dotnet.md, design/tech/python.md, design/tech/rust.md, design/tech/node-typescript.md, design/tech/sveltekit.md, design/tech/erlang-otp.md]
after: [T-0001]
usage:
  source: log
  seconds: 237
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 54
      output: 17443
      cache_read: 3233685
      cache_write: 62165
      cost: 1.4162
---
# T-0005 Write design/tech/ entries for every technology the sub-projects depend on

## Work

Write one file per technology under `design/tech/`, each with front matter (`title`, `updated`, `status`) and the version, where it is used, why it was chosen, and what was considered: `go.md`, `buf.md`, `grpc-protobuf.md`, `cockroachdb.md`, `spire.md` (SPIRE and go-spiffe), `helm.md`, `sops.md`, `dotnet.md`, `python.md`, `rust.md`, `node-typescript.md`, `sveltekit.md`, `erlang-otp.md`. Take each version from the sub-projects: `server/go.mod` and `client-go/go.mod`, `server/buf.gen.yaml` and the client `buf.gen.yaml` plugin pins, `proto/buf.yaml`, `server/deploy/helm`, `server/deploy/spire`, `server/Dockerfile`, `client-csharp/*.csproj`, `client-python/pyproject.toml`, `client-rust/Cargo.toml`, `client-typescript/package.json`, `admin/package.json`, and `client-erlang/README.md`. Say in `buf.md` how stubs are generated as T-0001 decided. Add a row per file to the table in `design/tech/README.md`. Where a version cannot be read from the tree, say so in the entry rather than guess.

## Done when

- `design/tech/` has the thirteen files above, each with version, where used, why chosen, and what was considered, and `design/tech/README.md` lists every one.
- No version is invented: each is traceable to a file in the tree or marked as not pinned.
- The markdown lint and `flai check --strict` are clean.

## Notes

Waits for T-0001 so that `buf.md` records the decided generation model. Runs alongside the release ADR task, with which it shares no path.
