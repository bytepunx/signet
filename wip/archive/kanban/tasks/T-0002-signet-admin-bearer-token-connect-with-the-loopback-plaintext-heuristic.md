---
id: T-0002
type: task
nature: feature
title: "Signet.Admin: bearer-token connect with the loopback-plaintext heuristic"
status: done
parent: S-0001
owner: alex
created: 2026-10-27T19:07:28Z
updated: 2026-10-27T19:12:35Z
transitions:
  - to: ready
    at: 2026-10-27T19:11:18Z
    by: agent
  - to: in-progress
    at: 2026-10-27T19:12:07Z
    by: agent
  - to: done
    at: 2026-10-27T19:12:35Z
    by: agent
stream: S-0001
tags: []
touches: [client-elixir/lib/signet/admin.ex, client-elixir/test/signet/admin_test.exs]
---
# T-0002 Signet.Admin: bearer-token connect with the loopback-plaintext heuristic

## Work

Implement `Signet.Admin.transport/2` (pure: address and options to `{:plaintext | {:tls, ssl_opts}}` or an error) and `Signet.Admin.connect/2`, mirroring the Go client's `DialAdmin`: loopback addresses plaintext, others TLS with the system store or `ca_pem`, `force_tls`, `plaintext`, mutual exclusions, empty-token rejection, and `authorization: Bearer <token>` as a channel header. Expose `admin_stub/0` and `gitops_stub/0` module references for callers.

## Done when

Behaviour tests cover every branch of the heuristic and the error cases without opening a socket, and `connect/2` succeeds against a plaintext loopback address only when a listener exists (tested via the error shape, not a real server).

## Notes
