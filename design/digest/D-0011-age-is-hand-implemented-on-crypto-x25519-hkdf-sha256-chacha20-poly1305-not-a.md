---
id: D-0011
title: "Age is hand-implemented on `:crypto` (X25519, HKDF-SHA256, ChaCha20-Poly1305), not a hex package, because none is maintained"
kind: decision
sources: [S-0003]
topics: [elixir, age, sops, client-elixir]
paths: [client-elixir/lib/signet/age.ex, client-elixir/lib/signet/age/bech32.ex, design/tech/age-elixir.md]
date: 2026-10-27
status: absorbed
absorbed_by: "design/tech/age-elixir.md#why"
asked_again: []
cited: []
---

# D-0011 Age is hand-implemented on `:crypto` (X25519, HKDF-SHA256, ChaCha20-Poly1305), not a hex package, because none is maintained

S-0003 implemented age on OTP `:crypto` (X25519, HKDF-SHA256 over HMAC, ChaCha20-Poly1305) rather than depend on a hex package: none is maintained, as the story notes anticipated.

Source: S-0003, `wip/archive/agents/S-0003.md`, Decisions. Recorded in `design/tech/age-elixir.md`.
