# Changelog

## Unreleased

### Features

* **elixir:** `Signet.Admin.connect/3`, bearer-token access to AdminService and GitOpsService with the Go client's loopback-plaintext heuristic
* **elixir:** `Signet.Workload.connect/3`, SPIFFE mTLS access to SecretsService with an X.509-SVID from the SPIRE Workload API, the identity-registration retry, and trust-domain authorisation of the server
* **elixir:** `Signet.Restart`, coordinated restarts: `watch_bundle/4`, `acquire_lock/5`, and `wait_for_restart/6` with the Go client's backoff, TTL/4 heartbeats, and one-message-per-burst change notifications
* **elixir:** stubs for `signet.v1`, `admin.v1`, and the SPIFFE Workload API generated with `protoc-gen-elixir` in a Docker toolchain
