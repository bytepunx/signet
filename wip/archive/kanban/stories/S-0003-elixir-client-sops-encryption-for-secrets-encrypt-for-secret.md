---
id: S-0003
type: story
nature: feature
title: "Elixir client: SOPS encryption for secrets (encrypt_for_secret)"
status: done
parent: E-0001
owner: alex
created: 2026-10-27T19:05:08Z
updated: 2026-10-27T22:31:00Z
transitions:
  - to: ready
    at: 2026-10-27T19:37:40Z
    by: alex
  - to: in-progress
    at: 2026-10-27T21:28:54Z
    by: agent-S-0003
  - to: review
    at: 2026-10-27T22:16:43Z
    by: agent-S-0003
  - to: done
    at: 2026-10-27T22:31:00Z
    by: alex
tags: []
topics: [elixir, client, sops, age]
touches: [client-elixir/lib/signet/sops_encrypt.ex, client-elixir/test/sops_encrypt_test.exs, client-elixir/lib/signet/age.ex, client-elixir/lib/signet/age/bech32.ex, client-elixir/test/signet/age_test.exs, client-elixir/test/support/age_test_support.ex, client-elixir/test/signet/sops_encrypt_test.exs, design/issues/I-0002-scripts-test-client-elixir-sh-rebuilds-the-toolchain-image-on-every-run-and-fails-when-the-registry-stalls.md, design/issues/summary.md, client-elixir/CHANGELOG.md, client-elixir/README.md, design/tech/README.md, design/tech/age-elixir.md, design/tech/sops.md, docs/users/elixir-client.md, client-elixir/Dockerfile, client-elixir/test/test_helper.exs, client-elixir/tests/integration/sops_encrypt_test.exs, scripts/README.md, scripts/integration-client-elixir.sh, scripts/integration.sh]
agent:
  harness: claude-code
usage:
  source: log
  seconds: 2888
  priced_by: harness
  turns:
    - day: 2026-10-27
      ceremony: 4
      hand_edits: 3
      work: 38
  models:
    - model: claude-fable-5-1
      input: 2546
      output: 90877
      cache_read: 11453105
      cache_write: 281284
      cost: 12.2202
---
# S-0003 Elixir client: SOPS encryption for secrets (encrypt_for_secret)

## Goal

`Signet.SopsEncrypt.encrypt_for_secret/2` produces the single-leaf, single-age-recipient SOPS YAML document the Go client's `EncryptForSecret` produces, so a workload can submit a value via `SyncBundle` that signet will actually persist.

## Acceptance criteria

- [x] Output decrypts with the real `sops` binary in an integration test run in Docker
- [x] AES-256-GCM leaf and MAC encryption with sops's 32-byte nonce and AAD matches the Go implementation
- [x] The data key is wrapped to the age recipient as an armored age file; the age implementation used is recorded in `design/tech/` with alternatives considered
- [x] Behaviour tests cover document shape and error cases, run by `make test`

## Tasks
- T-0010 Signet.SopsEncrypt.encrypt_for_secret/2 and its behaviour tests
- T-0011 Integration test: real sops decrypts the document, in Docker
- T-0012 Docs and tech records: age on :crypto, sops binary, README, user docs, changelog
- T-0018 Age X25519 recipient encryption on :crypto (Signet.Age)
- T-0019 Signet.SopsEncrypt.encrypt_for_secret/2 and its behaviour tests
- T-0020 Integration test: real sops decrypts the document, in Docker
- T-0021 Docs and tech records: age on :crypto, sops binary, README, user docs, changelog

## Notes

No mature age library exists on hex; expect to implement the age X25519 recipient format (X25519, HKDF-SHA256, ChaCha20-Poly1305) on `:crypto`, which supports all three primitives.

- 2026-10-27: `story/S-0003` and `story/S-0002` (both in review) change adjacent lines of `client-elixir/README.md`, `docs/users/elixir-client.md`, and `client-elixir/CHANGELOG.md`; the split is agreed in MS-0001 (S-0003 owns the SOPS section, the SOPS sentence, and one changelog bullet). Whichever is accepted second stops on those three paths at its rebase; the resolution is to keep both sections.
- The sops round-trip lives in `client-elixir/tests/integration/` and runs under `make integration` in Docker with sops 3.13.3; `make test` excludes it by tag. Verified here: 67 behaviour tests and 3 integration tests pass.
- T-0010 to T-0012 were duplicates from a failed first task-creation batch and were cancelled unworked.
