---
id: S-0001
type: story
nature: feature
title: "Elixir client: generated stubs, admin access, and SPIFFE workload access"
status: done
parent: E-0001
owner: alex
created: 2026-10-27T19:05:08Z
updated: 2026-10-27T19:37:26Z
transitions:
  - to: ready
    at: 2026-10-27T19:06:57Z
    by: agent
  - to: in-progress
    at: 2026-10-27T19:06:59Z
    by: agent
  - to: review
    at: 2026-10-27T19:24:48Z
    by: agent
  - to: done
    at: 2026-10-27T19:37:26Z
    by: alex
tags: [client-elixir]
topics: [elixir, client, spiffe, grpc]
touches: [client-elixir, design/tech, docs/users, scripts/test.sh, system-flow.yaml, scripts/README.md, scripts/test-client-elixir.sh, ".markdownlintignore", client-erlang/README.md]
---
# S-0001 Elixir client: generated stubs, admin access, and SPIFFE workload access

## Goal

A `client-elixir/` Mix package, `signet_client`, that mirrors the Go client's connection layer: stubs for `signet.v1` and `admin.v1` generated from the pinned signet-proto module, `Signet.Admin.connect/2` with the Go client's loopback-plaintext heuristic and bearer-token metadata, and `Signet.Workload.connect/3` that fetches an X.509-SVID from the SPIRE Workload API over its Unix socket, retries the "no identity issued" race on the Go client's schedule, and dials signet over mTLS authorising the server as a member of the expected trust domain. Built, generated, and tested inside Docker so no toolchain is installed on the host.

## Acceptance criteria

- [x] `client-elixir/` builds in Docker from a fresh clone with one script under `scripts/`, generating stubs with `protoc-gen-elixir` rather than committing them
- [x] `Signet.Admin.connect/2` picks plaintext for loopback addresses and TLS otherwise, honours `force_tls`, `plaintext`, and `ca_pem`, rejects the same conflicting combinations and empty tokens as the Go client, and attaches `authorization: Bearer <token>` to every RPC
- [x] `Signet.Workload.connect/3` fetches an SVID and trust bundle from the Workload API over a Unix socket, retries only "no identity issued" (gRPC `PermissionDenied`) with 1s, 2s, 4s, 8s backoff for five attempts total, and dials with mTLS that verifies the server's SPIFFE ID is in the trust domain with no hostname check
- [x] Behaviour tests cover the transport heuristic, the retry schedule against a fake Workload API, SPIFFE ID authorisation of the peer certificate, and bearer metadata, with no network and run by `make test`
- [x] `client-elixir/README.md`, `design/tech/`, and `docs/users/` describe the client, its dependencies, and how to build it
- [x] `flai check --strict` is clean

## Tasks
- T-0001 Scaffold client-elixir Mix project, Docker toolchain, and stub generation
- T-0002 Signet.Admin: bearer-token connect with the loopback-plaintext heuristic
- T-0003 Signet.Workload: SPIFFE X.509 source, identity retry, and mTLS connect
- T-0004 Document the Elixir client: README, example, design/tech, docs/users

## Notes

Stubs are gitignored and regenerated at build time, as the Python, Rust, and TypeScript clients do. Restart coordination and SOPS encryption are separate stories under E-0001.

### Planning

`client-elixir/test/support` holds the fakes and certificate helpers every test module shares, and `client-elixir/lib/signet/spiffe` the SPIFFE modules; later stories under E-0001 add to both as they need new fakes and SPIFFE helpers, so the folders stay in the story's claim.
