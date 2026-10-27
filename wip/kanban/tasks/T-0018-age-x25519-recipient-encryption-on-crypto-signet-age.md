---
id: T-0018
type: task
nature: feature
title: "Age X25519 recipient encryption on :crypto (Signet.Age)"
status: done
parent: S-0003
owner: arobson
created: 2026-10-27T21:31:39Z
updated: 2026-10-27T21:39:50Z
transitions:
  - to: ready
    at: 2026-10-27T21:31:57Z
    by: agent-S-0003
  - to: in-progress
    at: 2026-10-27T21:31:57Z
    by: agent-S-0003
  - to: done
    at: 2026-10-27T21:39:50Z
    by: agent-S-0003
stream: S-0003
tags: []
touches: [client-elixir/lib/signet/age.ex, client-elixir/lib/signet/age/bech32.ex, client-elixir/test/support/age_test_support.ex, client-elixir/test/signet/age_test.exs]
usage:
  source: log
  seconds: 473
  estimated: true
  priced_by: harness
  models:
    - model: claude-fable-5-1
      input: 465
      output: 16601
      cache_read: 2092174
      cache_write: 51383
      cost: 2.2323
---
# T-0018 Age X25519 recipient encryption on :crypto (Signet.Age)

## Work

Implement the age v1 X25519 recipient format on Erlang `:crypto`, enough to wrap a data key the way `filippo.io/age`'s `age.Encrypt` + `armor.NewWriter` does: Bech32 parsing of `age1...` recipients, an ephemeral X25519 key, HKDF-SHA256 wrap key, ChaCha20-Poly1305 file-key wrap, the header with its HMAC, the STREAM payload, and ASCII armor. Test support generates identities and decrypts, so behaviour tests round-trip without the age binary.

## Done when

- `Signet.Age.encrypt(recipient, plaintext)` returns `{:ok, armored}` for a valid recipient and `{:error, reason}` for an empty, malformed, or wrong-HRP recipient.
- Behaviour tests round-trip through the test-support decryptor and reject the wrong identity.
- `scripts/test-client-elixir.sh` passes with warnings as errors.

## Notes

Spec: the age format specification at <https://github.com/C2SP/C2SP/blob/main/age.md> (header, X25519 stanza, STREAM payload, armor).
