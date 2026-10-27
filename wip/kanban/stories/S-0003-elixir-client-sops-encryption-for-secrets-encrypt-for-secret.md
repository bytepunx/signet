---
id: S-0003
type: story
nature: feature
title: "Elixir client: SOPS encryption for secrets (encrypt_for_secret)"
status: backlog
parent: E-0001
owner: alex
created: 2026-10-27T19:05:08Z
updated: 2026-10-27T19:05:08Z
transitions: []
tags: []
topics: [elixir, client, sops, age]
touches: [client-elixir/lib/signet/sops_encrypt.ex, client-elixir/test/sops_encrypt_test.exs]
---
# S-0003 Elixir client: SOPS encryption for secrets (encrypt_for_secret)

## Goal

`Signet.SopsEncrypt.encrypt_for_secret/2` produces the single-leaf, single-age-recipient SOPS YAML document the Go client's `EncryptForSecret` produces, so a workload can submit a value via `SyncBundle` that signet will actually persist.

## Acceptance criteria

- [ ] Output decrypts with the real `sops` binary in an integration test run in Docker
- [ ] AES-256-GCM leaf and MAC encryption with sops's 32-byte nonce and AAD matches the Go implementation
- [ ] The data key is wrapped to the age recipient as an armored age file; the age implementation used is recorded in `design/tech/` with alternatives considered
- [ ] Behaviour tests cover document shape and error cases, run by `make test`

## Tasks

## Notes

No mature age library exists on hex; expect to implement the age X25519 recipient format (X25519, HKDF-SHA256, ChaCha20-Poly1305) on `:crypto`, which supports all three primitives.
