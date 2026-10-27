---
title: sops (test binary)
updated: 2026-10-27
status: active
topics: [client-elixir, sops]
---

# sops

| | |
|---|---|
| Version | 3.13.3, the `sops` binary from the getsops/sops GitHub release |
| Used in | `client-elixir/Dockerfile`, the test toolchain image only, for `client-elixir/tests/integration/sops_encrypt_test.exs`; the client never runs sops at runtime |
| Pinned in | The Dockerfile's `SOPS_VERSION` build argument |

## Why

`Signet.SopsEncrypt.encrypt_for_secret/2` hand-writes the single-leaf, single-recipient
slice of the SOPS file format, as the Go client's `EncryptForSecret` does. The integration
test, run by `make integration` in Docker, proves that document decrypts with the real
tool, not only with the client's own code, the same check the Go client's
`TestEncryptForSecret_RealSopsCanDecrypt` makes. It also checks that an identity the
document was not encrypted to cannot open it.

The document's `sops.version` field stays `3.13.1`, the value the Go client writes: sops
only requires it to be non-empty and does not gate decryption on its own build version.
The binary's version and the field's are independent; bump the binary when a release is
worth testing against, and the field only together with the Go client.

## Alternatives considered

- A Go test harness on the `github.com/getsops/sops/v3` module: pulls in every key
  backend (PGP, AWS, GCP, Azure, Vault) to test one age-only document.
- Trusting the behaviour tests' self-consistency alone: they decrypt with the client's own
  age decryptor and AES-GCM calls, so a drift from sops's format would pass unnoticed.
