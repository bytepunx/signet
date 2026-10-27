---
id: T-0070
type: task
nature: improvement
title: Replace the placeholder diagram in design/system/overview.md with the real parts
status: backlog
parent: S-0017
owner: alex
created: 2026-10-27T22:52:40Z
updated: 2026-10-27T22:52:40Z
transitions: []
stream: S-0017
tags: [docs]
touches: [design/system/overview.md]
---
# T-0070 Replace the placeholder diagram in design/system/overview.md with the real parts

## Work

Rewrite `## Parts` in `design/system/overview.md`. Replace the `Part A --> Part B` placeholder with a Mermaid diagram of the real parts and a short table naming each part, its folder, and its role:

- `server/` (signetd and the `signet` CLI), the service.
- `proto/`, the gRPC/protobuf schema (`SecretsService`, `AdminService`, `GitOpsService`), pushed to the Buf Schema Registry as `buf.build/bytepunx/signet-proto`.
- `admin/`, the read-only admin web UI (SvelteKit), a client of `AdminService` and `GitOpsService`.
- The workload clients: `client-go`, `client-python`, `client-typescript`, `client-rust`, `client-csharp`, `client-elixir`, `client-erlang`.

Show the relations: `proto/` is the schema every other part generates stubs from; the clients and `admin/` call the server over gRPC; the server talks to SPIRE, Kubernetes, and git. Take the facts from `server/design/draft.md` and each folder's README, and link `release.md` for how each part is released. Keep the Principles section. Bump `updated`.

Say in the text that all parts live in this one repository. Do not name `signet-clients`, `signet-proto`, or `signet-admin` as repositories.

## Done when

- `design/system/overview.md` shows server, proto, admin, and the seven clients and how they relate, in a Mermaid diagram that renders.
- No placeholder text remains in the file.
- `flai check --strict` and the markdown lint pass.

## Notes

Layer 1: it waits for no task and shares no path with another.
