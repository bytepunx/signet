---
title: System overview
updated: 2026-11-06
status: active
topics: [all]
---

# System overview

SPIFFE-based configuration and secrets distribution for Kubernetes

## Parts

signet is one mono-repo. Each part is a folder with its own build and tests.

```mermaid
flowchart LR
    proto["proto/<br>gRPC schema"] -->|generates stubs| server["server/<br>signetd and signet CLI"]
    proto -->|generates stubs| admin["admin/<br>SvelteKit admin UI"]
    proto -->|generates stubs| clients["client-*/<br>Go, Python, TypeScript,<br>Rust, C#, Elixir"]
    clients -->|SecretsService, mTLS with SVID| server
    admin -->|AdminService, GitOpsService| server
```

- `server/` is the service, `signetd`, and the `signet` operator CLI.
- `proto/` holds only `.proto` sources and buf configuration; every other part builds against it.
- `admin/` is a read-only operator web interface that talks to the server's admin listener.
- `client-*/` are the six client libraries workloads use to fetch secrets and watch for changes.

## Principles

- Files are the database.
- Defer detail until work is next up.
- Keep `wip/agents` current so any agent can resume.

## Decisions

- **The admin web interface is not a sub-project yet** (S-0018, 2026-11-06). `admin/` held a README and a CLAUDE.md and no code, and no story scheduled it, so a registered `sveltekit` component could be tagged and released empty (ADR-0001 retired `client-erlang` for the same reason). `admin` is removed from `system-flow.yaml`, `admin/` is deleted, and the plan is kept in [admin.md](admin.md). Scaffolding it is a separate epic.
