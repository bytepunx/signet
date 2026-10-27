---
id: T-0003
type: task
nature: feature
title: "Signet.Workload: SPIFFE X.509 source, identity retry, and mTLS connect"
status: done
parent: S-0001
owner: alex
created: 2026-10-27T19:07:28Z
updated: 2026-10-27T19:20:40Z
transitions:
  - to: ready
    at: 2026-10-27T19:12:36Z
    by: agent
  - to: in-progress
    at: 2026-10-27T19:14:34Z
    by: agent
  - to: done
    at: 2026-10-27T19:20:40Z
    by: agent
stream: S-0001
tags: []
touches: [client-elixir/lib/signet/workload.ex, client-elixir/lib/signet/spiffe, client-elixir/test/signet/workload_test.exs, client-elixir/test/support, client-elixir/lib/signet/admin.ex, client-elixir/mix.exs, client-elixir/test/signet/admin_test.exs, client-elixir/test/signet/spiffe/authorize_test.exs, client-elixir/test/signet/spiffe/trust_domain_test.exs, client-elixir/test/signet/spiffe/x509_context_test.exs, client-elixir/test/signet/spiffe/x509_source_test.exs]
---
# T-0003 Signet.Workload: SPIFFE X.509 source, identity retry, and mTLS connect

## Work

Implement `Signet.Spiffe.TrustDomain` (parse and validate), `Signet.Spiffe.Authorize` (an Erlang `ssl` `verify_fun` that accepts a peer whose SPIFFE URI SAN is in the expected trust domain), `Signet.Spiffe.X509Source` (a GenServer holding the current SVID and bundle from the Workload API `FetchX509SVID` stream over `unix://`, with the `workload.spiffe.io: true` header), the single-shot probe with the Go client's 1s, 2s, 4s, 8s retry on `PermissionDenied` only, and `Signet.Workload.connect/3` that dials signet with `cert`, `key`, `cacerts`, `verify_peer`, no SNI or hostname check, and the authoriser.

## Done when

Behaviour tests cover the retry schedule and classification with a fake Workload API behind a behaviour, the authoriser against self-signed certificates with and without SPIFFE SANs, trust-domain parsing, and the TLS option shape, all without a network.

## Notes
