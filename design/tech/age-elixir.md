---
title: age on OTP crypto (Elixir client)
updated: 2026-10-27
status: active
topics: [client-elixir, sops, age]
---

# age on OTP crypto

| | |
|---|---|
| Version | age format v1 (the X25519 recipient stanza, header MAC, STREAM payload, and ASCII armor), hand-implemented; no package |
| Used in | `client-elixir/lib/signet/age.ex`: wrapping the SOPS data key `Signet.SopsEncrypt` produces |
| Pinned in | Nothing to pin: every primitive comes from OTP `:crypto` (Erlang/OTP 28.5, see [elixir.md](elixir.md)) |

## Why

No maintained age library is published on hex.pm as of 2026-10-27, as the story that
added this anticipated. The slice of age the client needs is small: one X25519 recipient,
which takes X25519 key agreement, HKDF-SHA256 (RFC 5869, built on `:crypto.mac/4`'s
HMAC, since OTP ships no HKDF), and ChaCha20-Poly1305 for the file-key wrap and the
payload chunks. OTP `:crypto` has all three, so `Signet.Age` is a few dozen lines on the
standard library and `Signet.Age.Bech32` decodes the `age1...` recipient string.

Only encryption is implemented: the client wraps a data key to signet's recipient and
never decrypts. The decryptor that the behaviour tests round-trip through lives in
`client-elixir/test/support/age_test_support.ex`, strict enough to catch format mistakes,
and is not part of the library. The wire format was checked against the real `age` binary
while the module was written, and `tests/integration/sops_encrypt_test.exs` has the real
`sops` binary ([sops.md](sops.md)) decrypt the SOPS document whose data key it wraps.

## Alternatives considered

- A hex package for age: none is published on hex.pm, maintained or otherwise, as of
  2026-10-27.
- A Rust NIF around `rage` (the Rust age implementation): adds a native build step and a
  Rust toolchain to a pure-Elixir library, for three primitives OTP already has.
- Shelling out to the `age` binary: a runtime dependency on a host binary in every
  workload image that uses the client, where the client otherwise needs only the BEAM.
