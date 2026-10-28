# scripts

Purpose-named shell scripts for common tasks. The `Makefile` calls these; CI calls the same ones. Each script is safe to run from any directory, prints what it does, and exits non-zero on failure.

| Script | Does |
|--------|------|
| `check.sh` | Runs `flai check --strict` on this repository |
| `test.sh` | Behavior tests, fast, every iteration (`make test`); runs every component's test script, all of them even when one fails, and names the failed ones |
| `buf-generate.sh` | Runs `buf generate` for one component inside the pinned `bufbuild/buf` image; the test scripts of the generating components call it |
| `check-server-chart.sh` | Helm lint and render checks for the server chart, inside Docker; `test-server.sh` calls it |
| `test-server.sh` | Server `go test -race`, `golangci-lint`, `make build`, stub drift, and the chart checks, inside Docker |
| `test-proto.sh` | `buf lint` and `buf breaking` against the last `proto/v*` tag (skipped when there is none), inside Docker |
| `test-client-go.sh` | Go client stub drift, `go vet`, and `go test -race`, inside Docker |
| `test-client-python.sh` | Python client stub generation and `pytest`, inside Docker |
| `test-client-typescript.sh` | TypeScript client stub drift, `npm ci`, build, and tests, inside Docker |
| `test-client-rust.sh` | Rust client stub generation and `cargo test`, inside Docker |
| `test-client-csharp.sh` | C# client stub generation and `dotnet test`, inside Docker |
| `test-client-elixir.sh` | Builds the Elixir client's toolchain image and runs stub generation, compile, and `mix test` inside Docker; any other `mix` or `buf` command can be passed as arguments |
| `integration.sh` | Integration tests against real adapters (`make integration`); runs every component's integration script |
| `integration-client-elixir.sh` | Builds the Elixir client's toolchain image and runs `client-elixir/tests/integration/` inside Docker, where the real `sops` binary decrypts what `Signet.SopsEncrypt` produces |
| `set-release-secrets.sh` | `check` (default) lists which release secrets of `bytepunx/signet` are set; `set` sets them from environment variables of the same name (operator) |
| `protect-main.sh` | Dry run of the branch protection on `main` with the CI jobs as required checks; `--apply` sets it, `--show` prints it (operator) |
| `archive-old-repos.sh` | Dry run of pointing the three old repositories here and archiving them; `--apply` does it (operator) |
| `smoke.sh` | End-to-end smoke tests (`make smoke`) |

Docker and network access to the Buf Schema Registry and the package registries are the only prerequisites of the `test-*.sh` scripts. `make test-<component>` runs one of them, such as `make test-client-go`.
