## signet-admin

Read-only (from a data-exposure standpoint — never displays secret values) admin web
interface for signet, a configuration and secrets management service for Kubernetes.

### Motivation

Give operators a way to see a signet deployment's state — secrets metadata, config
content, repos, policies, KEKs, SOPS keys, seal status, audit history — without needing
the `signet` CLI or direct database access for every question.

### Relationship to bytepunx/signet

This repo is a separate deployable frontend/BFF, not part of signet's Go module. It
consumes signet's `AdminService`/`GitOpsService` gRPC surface exclusively — no direct
database access, no duplicated business logic. Any backend capability this UI needs that
doesn't exist yet (a list/read RPC, a query) should be filed and built in
`bytepunx/signet`, not worked around here.

### Protocol

The gRPC/protobuf schema lives in [bytepunx/signet-proto](https://github.com/bytepunx/signet-proto),
distributed via the Buf Schema Registry (`buf.build/bytepunx/signet-proto`). This repo
generates its own TypeScript client stubs from that same module — see whatever
`buf.gen.yaml` (or equivalent) is checked in here once code exists.

### Design documentation

Once implementation begins, design decisions should be tracked the same way
`bytepunx/signet` does it — under a `design/` folder, kept in sync with the code.
