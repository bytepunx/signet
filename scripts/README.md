# scripts

Purpose-named shell scripts for common tasks. The `Makefile` calls these; CI calls the same ones. Each script is safe to run from any directory, prints what it does, and exits non-zero on failure.

| Script | Does |
|--------|------|
| `check.sh` | Runs `flai check --strict` on this repository |
| `test.sh` | Behavior tests, fast, every iteration (`make test`); runs every component's test script |
| `test-client-elixir.sh` | Builds the Elixir client's toolchain image and runs stub generation, compile, and `mix test` inside Docker; any other `mix` or `buf` command can be passed as arguments |
| `integration.sh` | Integration tests against real adapters (`make integration`); runs every component's integration script |
| `integration-client-elixir.sh` | Builds the Elixir client's toolchain image and runs `client-elixir/tests/integration/` inside Docker, where the real `sops` binary decrypts what `Signet.SopsEncrypt` produces |
| `smoke.sh` | End-to-end smoke tests (`make smoke`) |
