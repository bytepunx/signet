---
title: Admin web interface (planned)
updated: 2026-11-06
status: draft
topics: [admin, sveltekit]
---

# Admin web interface (planned)

Not started. This note keeps the plan from the former `admin/` folder, which held a README and a CLAUDE.md and no code. The `admin` sub-project was removed from `system-flow.yaml` until work begins; see [System overview](overview.md#decisions).

## Purpose

A read-only admin web interface for signet. It gives operators a browsable view of a deployment's state without the `signet` CLI or direct database access: secret metadata (names, versions, timestamps, never values), full config document content, registered repositories, access policies, KEKs, SOPS keys, seal status, and paginated audit log history. It never displays secret values.

## Architecture

- [SvelteKit](https://kit.svelte.dev/), deployed as its own service alongside signet.
- Backend-for-frontend: SvelteKit's server-side routes speak gRPC to signetd's admin listener (`admin.clusterAccess` plus `admin.tls`). The browser talks only same-origin HTTP/JSON to SvelteKit, never gRPC.
- Auth: an operator pastes a Kubernetes ServiceAccount token (the one `kubectl create token` produces and the `signet` CLI accepts). It is stored as an httpOnly session cookie and forwarded as the bearer token on every backend call. This mirrors signet's admin trust model: no new identity system, no OIDC (yet). Each operator's actions are attributed to them in signet's audit log, as with the CLI.
- The UI acts with full admin privilege (the same `SIGNET_ADMIN_SUBJECTS`/RBAC model as the CLI). It is an operator tool, not a general-audience surface.

## Relationship to the rest of the repository

- A separate deployable frontend and BFF, not part of the Go module. It consumes the `AdminService` and `GitOpsService` gRPC surface only: no direct database access, no duplicated business logic.
- A backend capability the UI needs that does not exist yet (a list or read RPC, a query) is built in `server/`, not worked around in the UI.
- TypeScript client stubs are generated from the in-repo `proto/` module with its own `buf.gen.yaml`, as the other clients do.

## Starting the work

Scaffold it under its own epic, then add `admin` back to `system-flow.yaml` with `kind: sveltekit` and a `path` of `admin`, so the release flow can tag it from the first real version.
