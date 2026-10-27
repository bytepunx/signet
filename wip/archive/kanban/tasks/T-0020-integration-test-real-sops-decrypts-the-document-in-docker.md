---
id: T-0020
type: task
nature: feature
title: "Integration test: real sops decrypts the document, in Docker"
status: done
parent: S-0003
owner: arobson
created: 2026-10-27T21:31:57Z
updated: 2026-10-27T22:15:14Z
transitions:
  - to: ready
    at: 2026-10-27T21:31:57Z
    by: agent-S-0003
  - to: in-progress
    at: 2026-10-27T22:03:33Z
    by: agent-S-0003
  - to: done
    at: 2026-10-27T22:15:14Z
    by: agent-S-0003
stream: S-0003
tags: []
touches: [client-elixir/Dockerfile, client-elixir/tests/integration/sops_encrypt_test.exs, scripts/integration-client-elixir.sh, scripts/integration.sh, scripts/README.md, client-elixir/test/test_helper.exs]
after: [T-0019]
usage:
  source: log
  seconds: 701
  estimated: true
  priced_by: harness
  models:
    - model: claude-fable-5-1
      input: 299
      output: 10668
      cache_read: 1344496
      cache_write: 33020
      cost: 1.4345
---
# T-0020 Integration test: real sops decrypts the document, in Docker

## Work

Install the pinned `sops` binary (3.13.3) in the toolchain image, add `client-elixir/tests/integration/sops_encrypt_test.exs` that encrypts to a generated age identity, writes the document to a temp dir, and runs `sops --decrypt` with `SOPS_AGE_KEY`, both for the right identity (value round-trips) and the wrong one (decrypt fails). Add `scripts/integration-client-elixir.sh` that runs it in Docker and wire it into `scripts/integration.sh` so `make integration` runs it.

## Done when

- `make integration` runs the test in Docker and passes.
- `make test` still runs behaviour tests only.
- `scripts/README.md` lists the new script.

## Notes

The sops release is pinned by version in the Dockerfile and recorded in `design/tech/sops.md` by the docs task.
