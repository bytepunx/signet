# signet-admin

**Read-only admin web interface for [signet](https://github.com/bytepunx/signet).**

signet-admin gives operators a browsable view of a signet deployment's state — secret
metadata (names, versions, timestamps — never values), full config document content,
registered repositories, access policies, KEKs, SOPS keys, seal status, and paginated
audit log history. It never displays secret values.

## Status

Pre-implementation. See open issues for planned work.

## Architecture (planned)

- [SvelteKit](https://kit.svelte.dev/), deployed as its own service alongside signet.
- A backend-for-frontend pattern: SvelteKit's server-side routes speak real gRPC to
  signetd's admin listener (via `admin.clusterAccess` + `admin.tls` — see signet's
  `design/draft.md` Section 10); the browser only ever talks same-origin HTTP/JSON to
  SvelteKit itself, never gRPC directly.
- Auth: an operator logs in by pasting a Kubernetes ServiceAccount token (the same one
  `kubectl create token` produces, and the same one the `signet` CLI already accepts),
  stored as an httpOnly session cookie and forwarded as the bearer token on every
  backend call. This mirrors signet's existing admin trust model exactly — no new
  identity system, no OIDC integration (yet). Every action a given operator takes is
  attributed to them individually in signet's own audit log, the same as CLI use.
- The UI authenticates with full admin privilege (same `SIGNET_ADMIN_SUBJECTS`/RBAC
  model as the CLI) — it is an operator tool, not a general-audience surface.

## Related repos

- [bytepunx/signet](https://github.com/bytepunx/signet) — the service this UI fronts.
- [bytepunx/signet-proto](https://github.com/bytepunx/signet-proto) — gRPC/protobuf
  schema (`AdminService`, `GitOpsService`, `SecretsService`); this repo generates its own
  TypeScript client stubs from the same module.
