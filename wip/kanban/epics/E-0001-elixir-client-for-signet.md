---
id: E-0001
type: epic
nature: feature
title: Elixir client for signet
status: review
owner: alex
created: 2026-10-27T19:04:34Z
updated: 2026-10-27T22:16:43Z
transitions:
  - to: ready
    at: 2026-10-27T19:06:57Z
    by: agent
  - to: in-progress
    at: 2026-10-27T19:06:59Z
    by: agent
  - to: review
    at: 2026-10-27T22:16:43Z
    by: agent-S-0003
tags: []
topics: [elixir, client, spiffe]
touches: [client-elixir]
usage:
  source: sum
  seconds: 5750
  priced_by: harness
  turns:
    - day: 2026-10-27
      ceremony: 5
      test_runs: 2
      hand_edits: 7
      work: 91
  models:
    - model: claude-fable-5-1
      input: 5266
      output: 193358
      cache_read: 28914598
      cache_write: 642084
      cost: 28.2228
---
# E-0001 Elixir client for signet

## Outcome

An Elixir client library, `client-elixir/`, with the same capabilities as the Go client: SPIFFE mTLS workload access to SecretsService via the SPIRE Workload API, bearer-token admin access to AdminService and GitOpsService, coordinated restarts (bundle watch and restart lock), and client-side SOPS encryption for secrets. Built and tested in Docker, generated from the pinned signet-proto module.

## Stories
- S-0001 Elixir client: generated stubs, admin access, and SPIFFE workload access
- S-0002 Elixir client: coordinated restarts (bundle watch and restart lock)
- S-0003 Elixir client: SOPS encryption for secrets (encrypt_for_secret)

## Notes

The Erlang client folder (`client-erlang/`) records why Buf's remote plugins cannot generate BEAM stubs. The Elixir client takes the local-toolchain path that README anticipated: `protoc-gen-elixir` from the `protobuf` hex package, run inside the Docker image the client builds and tests in.
- 2026-10-27T19:06:57Z: moved to ready: follows S-0001, which moved to ready
- 2026-10-27T19:06:59Z: moved to in-progress: follows S-0001, which moved to in-progress
- 2026-10-27T22:16:43Z: moved to review: follows S-0003, which moved to review
