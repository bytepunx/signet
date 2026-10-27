---
id: D-0014
title: "The sops round-trip is an integration test under `tests/integration/`, run by `make integration` in Docker with a pinned sops binary; `make test` stays free of it"
kind: decision
sources: [S-0003]
topics: [elixir, sops, test, client-elixir]
paths: [client-elixir/tests/integration/sops_encrypt_test.exs, client-elixir/Dockerfile, design/tech/sops.md]
date: 2026-10-27
status: absorbed
absorbed_by: "design/tech/sops.md#why"
asked_again: []
cited: []
---

# D-0014 The sops round-trip is an integration test under `tests/integration/`, run by `make integration` in Docker with a pinned sops binary; `make test` stays free of it

The real-sops round-trip is an integration test under `client-elixir/tests/integration/`, run by `make integration` in Docker with a pinned sops binary (3.13.3), as `code-quality.md` places integration tests. `make test` stays free of it.

Source: S-0003, `wip/archive/agents/S-0003.md`, Decisions. Recorded in `design/tech/sops.md`.
