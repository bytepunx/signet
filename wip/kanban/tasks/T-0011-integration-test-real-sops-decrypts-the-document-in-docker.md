---
id: T-0011
type: task
nature: feature
title: "Integration test: real sops decrypts the document, in Docker"
status: cancelled
parent: S-0003
owner: arobson
created: 2026-10-27T21:31:23Z
updated: 2026-10-27T22:16:04Z
transitions:
  - to: cancelled
    at: 2026-10-27T22:16:04Z
    by: agent-S-0003
stream: S-0003
tags: []
touches: [client-elixir/Dockerfile, client-elixir/tests/integration/sops_encrypt_test.exs, scripts/integration-client-elixir.sh, scripts/integration.sh, scripts/README.md]
---
# T-0011 Integration test: real sops decrypts the document, in Docker

## Work

Install the pinned `sops` binary (3.13.3) in the toolchain image, add `client-elixir/tests/integration/sops_encrypt_test.exs` that encrypts to a generated age identity, writes the document to a temp dir, and runs `sops --decrypt` with `SOPS_AGE_KEY`, both for the right identity (value round-trips) and the wrong one (decrypt fails). Add `scripts/integration-client-elixir.sh` that runs it in Docker and wire it into `scripts/integration.sh` so `make integration` runs it.

## Done when

- `make integration` runs the test in Docker and passes.
- `make test` still runs behaviour tests only.
- `scripts/README.md` lists the new script.

## Notes

The sops release is pinned by version in the Dockerfile and recorded in `design/tech/sops.md` (T-0004).
- 2026-10-27T22:16:04Z: moved to cancelled: duplicate of T-0019/T-0020/T-0021: created by a first task-creation attempt whose output parsing failed; never worked
