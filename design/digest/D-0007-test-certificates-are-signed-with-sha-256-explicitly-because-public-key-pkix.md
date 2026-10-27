---
id: D-0007
title: "Test certificates are signed with SHA-256 explicitly, because `:public_key.pkix_test_data` defaults to SHA-1"
kind: cause
sources: [S-0001]
topics: [elixir, client-elixir, spiffe, test]
paths: [client-elixir/test]
date: 2026-10-27
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0007 Test certificates are signed with SHA-256 explicitly, because `:public_key.pkix_test_data` defaults to SHA-1

`:public_key.pkix_test_data` signs with SHA-1 by default. TLS 1.3 refuses such a certificate with an unhelpful `unable_to_supply_acceptable_cert`. Tests that handshake over loopback pass SHA-256 explicitly.

Source: S-0001, `wip/archive/agents/S-0001.md`, Decisions.
