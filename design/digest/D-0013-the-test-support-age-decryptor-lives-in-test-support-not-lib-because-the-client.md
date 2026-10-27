---
id: D-0013
title: "The test-support age decryptor lives in `test/support`, not `lib`, because the client never decrypts"
kind: decision
sources: [S-0003]
topics: [elixir, age, test, client-elixir]
paths: [client-elixir/test/support/age_test_support.ex, design/tech/age-elixir.md]
date: 2026-10-27
status: absorbed
absorbed_by: "design/tech/age-elixir.md#why"
asked_again: []
cited: []
---

# D-0013 The test-support age decryptor lives in `test/support`, not `lib`, because the client never decrypts

The age decryptor exists so behaviour tests can round-trip a SOPS document without an age binary. The client never decrypts, so it stays in `client-elixir/test/support/` and is not part of the library.

Source: S-0003, `wip/archive/agents/S-0003.md`, Decisions. Recorded in `design/tech/age-elixir.md`.
