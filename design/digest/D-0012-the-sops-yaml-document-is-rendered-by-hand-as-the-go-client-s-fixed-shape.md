---
id: D-0012
title: The SOPS YAML document is rendered by hand, as the Go client's fixed shape allows, rather than adding a YAML dependency
kind: decision
sources: [S-0003]
topics: [elixir, sops, client-elixir]
paths: [client-elixir/lib/signet/sops_encrypt.ex]
date: 2026-10-27
status: absorbed
absorbed_by: "design/tech/sops.md#why"
asked_again: []
cited: []
---

# D-0012 The SOPS YAML document is rendered by hand, as the Go client's fixed shape allows, rather than adding a YAML dependency

S-0003 renders the SOPS-encrypted YAML document in `Signet.SopsEncrypt` by hand. The Go client's document has a fixed shape (single leaf, single age recipient), so no YAML dependency is added.

Source: S-0003, `wip/archive/agents/S-0003.md`, Decisions.
