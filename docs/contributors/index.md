---
title: Contributors guide
updated: 2026-11-06
status: active
---

# signet for contributors

signet is one mono-repo of nine components. Each is a folder with its own build, and each has a `make` target that runs its tests the same way CI does. Docker, plus network access to the Buf Schema Registry and the package registries, are the only prerequisites; every test script runs its toolchain in a container.

Run `make test` for every component, or `make test-<component>` for one. [`scripts/README.md`](../../scripts/README.md) lists every script.

## Components

| Component | What it is | Build and test | More |
|-----------|-----------|----------------|------|
| `server/` | `signetd` and the `signet` CLI, in Go | `make test-server`: `go test -race`, `golangci-lint`, build, stub drift, Helm chart checks. From `server/`, `make build`, `make test`, `make lint` | [README](../../server/README.md) |
| `proto/` | The gRPC schema and buf configuration, no generated code | `make test-proto`: `buf lint`, and `buf breaking` against the last `proto/v*` tag | [README](../../proto/README.md) |
| `admin/` | Read-only SvelteKit admin UI, not yet implemented | No build or tests yet | [README](../../admin/README.md) |
| `client-go/` | Go client | `make test-client-go`: stub drift, `go vet`, `go test -race` | [README](../../client-go/README.md) |
| `client-python/` | Python client | `make test-client-python`: stub generation, `pytest` | [README](../../client-python/README.md) |
| `client-typescript/` | TypeScript client for Node | `make test-client-typescript`: stub drift, `npm ci`, build, tests | [README](../../client-typescript/README.md) |
| `client-rust/` | Rust client | `make test-client-rust`: stub generation, `cargo test` | [README](../../client-rust/README.md) |
| `client-csharp/` | C# client | `make test-client-csharp`: stub generation, `dotnet test` | [README](../../client-csharp/README.md) |
| `client-elixir/` | Elixir client | `make test-client-elixir`: builds the toolchain image once, then stub generation, compile, `mix test` | [README](../../client-elixir/README.md) |

## Schema changes

Every component builds against `proto/`. After you change a `.proto` file, regenerate the stubs of each component that commits them (`scripts/buf-generate.sh <component>`); the test scripts of `server/`, `client-go/`, and `client-typescript/` fail when the committed stubs have drifted.

## Git hooks

From `server/`, `make install-hooks` points git at `server/githooks`. Its pre-commit hook runs `gofmt` and `golangci-lint` on `server/` before each commit, the same checks as CI's Lint job. Skip it once with `git commit --no-verify`.

## Design and work

The design is in [`design/system/`](../../design/system/overview.md). Work items, the board, and agent narratives are under `wip/`; `make check` validates the whole repository and `make lint-md` lints the markdown.
