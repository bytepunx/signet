---
id: T-0010
type: task
nature: feature
title: Signet.SopsEncrypt.encrypt_for_secret/2 and its behaviour tests
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
touches: [client-elixir/lib/signet/sops_encrypt.ex, client-elixir/test/signet/sops_encrypt_test.exs]
---
# T-0010 Signet.SopsEncrypt.encrypt_for_secret/2 and its behaviour tests

## Work

Port `client-go/sops_encrypt.go` to `Signet.SopsEncrypt.encrypt_for_secret/2`: a random 32-byte data key, AES-256-GCM leaves with sops's 32-byte nonce and AAD (`value:` for the leaf, `lastmodified` for the MAC), the SHA-512 uppercase-hex MAC, the data key wrapped with `Signet.Age`, and a hand-rendered YAML document with the shape the Go client emits. No YAML dependency.

## Done when

- `encrypt_for_secret(public_key, value)` returns `{:ok, yaml}` whose `value`, `sops.age[0].recipient`, `sops.age[0].enc`, `sops.lastmodified`, `sops.mac`, and `sops.version` match the Go document's shape.
- Empty and malformed public keys return `{:error, reason}`.
- Behaviour tests check the document shape, that the leaf and MAC decrypt with the data key recovered through the test-support age decryptor, and the error cases.
- `scripts/test-client-elixir.sh` passes with warnings as errors.

## Notes

The Go implementation is the reference, verified against sops 3.x's `Tree.Encrypt` and `aes.Cipher.Encrypt`.
- 2026-10-27T22:16:04Z: moved to cancelled: duplicate of T-0019/T-0020/T-0021: created by a first task-creation attempt whose output parsing failed; never worked
