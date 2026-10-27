---
id: D-0005
title: "Server authorisation in the Elixir client is a `verify_fun` on the single SPIFFE URI SAN's trust domain with SNI disabled"
kind: decision
sources: [S-0001]
topics: [elixir, client-elixir, spiffe, grpc]
paths: [client-elixir/lib/signet/spiffe, design/tech/elixir.md]
date: 2026-10-27
status: absorbed
absorbed_by: "design/tech/elixir.md#why"
asked_again: []
cited: []
---

# D-0005 Server authorisation in the Elixir client is a `verify_fun` on the single SPIFFE URI SAN's trust domain with SNI disabled

Authorisation is a `verify_fun` that checks the leaf's single SPIFFE URI SAN is in the expected trust domain, with `server_name_indication: :disable`. It mirrors go-spiffe's `AuthorizeMemberOf`, and is what the Python client could not do because grpc-python has no such hook. An X.509-SVID has no DNS or IP name, so the hostname check would always fail.

Source: S-0001, `wip/archive/agents/S-0001.md`, Decisions.
