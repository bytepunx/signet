---
title: Elixir client
updated: 2026-10-27
status: active
topics: [client-elixir]
---

# Elixir client

`signet_client` lets an Elixir or Erlang service read its secrets and configuration from
signet over SPIFFE mTLS, and lets operator tooling reach the admin API with a bearer
token. The package lives in [`client-elixir/`](../../client-elixir/README.md), whose
README is the reference; this page is the short path.

## Reading a secret from a workload

```elixir
{:ok, channel, source} =
  Signet.Workload.connect("signet.internal:8443",
    "unix:///run/spire/sockets/agent.sock", "example.org")

{:ok, %{value: value}} =
  Signet.V1.SecretsService.Stub.get_secret(channel, %Signet.V1.GetSecretRequest{
    namespace: "default", service: "example", name: "api-key"
  })
```

The three arguments are signet's workload address, the SPIRE agent's Workload API socket
mounted into the pod, and the trust domain signet's own SVID belongs to. The client
fetches the pod's X.509-SVID from the socket, waits out the few seconds a brand-new pod's
identity can take to be registered, and verifies that the server it reaches presents a
SPIFFE ID in that trust domain. Keep `source` for as long as the channel is open and
pass both to `Signet.Workload.disconnect/2` afterwards.

## Operator access

```elixir
{:ok, channel} = Signet.Admin.connect("localhost:8444", token)
```

A loopback address, as with `kubectl port-forward`, is plaintext; any other address is
TLS. `ca_pem:`, `force_tls:`, and `plaintext:` adjust that, as the README describes.

## Coordinated restarts

```elixir
{:ok, lock} = Signet.Restart.wait_for_restart(channel, "default", "example", 30_000, 10_000)
# graceful shutdown: drain in-flight requests, close resources
:ok = Signet.Restart.release(lock)
System.stop(0)
```

A service fetches its bundle at startup, serves traffic, and blocks on `wait_for_restart/6`
until signet reports a change for its namespace and service and this replica holds the
restart lock, so at most one replica restarts at a time. It then does its own graceful
shutdown, releases the lock for the next replica, and exits; Kubernetes starts a fresh
process that fetches the new bundle. The two durations are milliseconds: the lock TTL,
which must cover the shutdown, and the debounce that absorbs rapid successive changes.

## Not yet available

Client-side SOPS encryption is planned; the Go client has it today.
