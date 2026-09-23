## signet

Signet is a configuration and secrets management service that provides a secure means of distributing service information and event-driven updates for Kubernetes clusters.

### Motivation

A complete and secure solution that uses SPIFFE/SPIRE for service identity based configuration and secret retrieval.

### Design

All design documentation is under the design folder. All changes must be reflected in the design documentation.

### Protocol

The gRPC/protobuf schema (`SecretsService`, `AdminService`, `GitOpsService`) lives in the
separate [bytepunx/signet-proto](https://github.com/bytepunx/signet-proto) repository, not
in this one. It's distributed via the Buf Schema Registry
(`buf.build/bytepunx/signet-proto`); `buf.gen.yaml` here generates this repo's own `gen/`
stubs from that module (see the `inputs:` block). To pick up schema changes, edit the
module reference in `buf.gen.yaml` (pin to a specific `:vX.Y.Z` label once tagged releases
exist) and run `make proto`. Client implementations live in a third repo,
[bytepunx/signet-clients](https://github.com/bytepunx/signet-clients).

