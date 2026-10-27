---
id: T-0019
type: task
nature: feature
title: Signet.SopsEncrypt.encrypt_for_secret/2 and its behaviour tests
status: done
parent: S-0003
owner: arobson
created: 2026-10-27T21:31:57Z
updated: 2026-10-27T21:56:00Z
transitions:
  - to: ready
    at: 2026-10-27T21:31:57Z
    by: agent-S-0003
  - to: in-progress
    at: 2026-10-27T21:40:02Z
    by: agent-S-0003
  - to: done
    at: 2026-10-27T21:56:00Z
    by: agent-S-0003
stream: S-0003
tags: []
touches: [client-elixir/lib/signet/sops_encrypt.ex, client-elixir/test/signet/sops_encrypt_test.exs]
after: [T-0018]
usage:
  source: log
  seconds: 958
  estimated: true
  priced_by: harness
  models:
    - model: claude-fable-5-1
      input: 600
      output: 21431
      cache_read: 2700970
      cache_write: 66335
      cost: 2.8819
---
# T-0019 Signet.SopsEncrypt.encrypt_for_secret/2 and its behaviour tests

## Work

Port `client-go/sops_encrypt.go` to `Signet.SopsEncrypt.encrypt_for_secret/2`: a random 32-byte data key, AES-256-GCM leaves with sops's 32-byte nonce and AAD (`value:` for the leaf, `lastmodified` for the MAC), the SHA-512 uppercase-hex MAC, the data key wrapped with `Signet.Age`, and a hand-rendered YAML document with the shape the Go client emits. No YAML dependency.

## Done when

- `encrypt_for_secret(public_key, value)` returns `{:ok, yaml}` whose `value`, `sops.age[0].recipient`, `sops.age[0].enc`, `sops.lastmodified`, `sops.mac`, and `sops.version` match the Go document's shape.
- Empty and malformed public keys return `{:error, reason}`.
- Behaviour tests check the document shape, that the leaf and MAC decrypt with the data key recovered through the test-support age decryptor, and the error cases.
- `scripts/test-client-elixir.sh` passes with warnings as errors.

## Notes

The Go implementation is the reference, verified against sops 3.x's `Tree.Encrypt` and `aes.Cipher.Encrypt`.
