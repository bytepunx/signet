## signet-proto

The protobuf schema for the [signet](https://github.com/bytepunx/signet) secrets and
configuration management protocol. This repository contains **only** `.proto` sources and
buf configuration — no generated code is committed or published here.

### What's in here

```
proto/
  signet/v1/secrets.proto   # SecretsService — secret + config retrieval, watch, restart locks
  admin/v1/admin.proto      # AdminService / GitOpsService — unseal, KEK/SOPS rotation, GitOps sync
```

### Consuming this schema

The schema is published to the [Buf Schema Registry](https://buf.build/bytepunx/signet-proto).
Consumers (the `signet` server, and each language client in
[signet-clients](https://github.com/bytepunx/signet-clients)) declare it as a dependency in
their own `buf.yaml` and generate their own language stubs locally:

```yaml
# buf.yaml
version: v2
deps:
  - buf.build/bytepunx/signet-proto
```

Run `buf dep update` to pin (or bump) the exact schema version in your `buf.lock`, then
`buf generate` to produce stubs into your own repo. Each consumer controls its own package
layout — none of the proto files hard-code a `go_package` (or equivalent) — via buf's
[managed mode](https://buf.build/docs/generate/managed-mode/) in their own `buf.gen.yaml`.

### Versioning and breaking changes

Releases are tagged `vX.Y.Z` (via release-please) and pushed to BSR under a matching label,
so consumers can pin to an explicit, human-readable version instead of a floating branch
head. `buf breaking` runs in CI against the last tagged release on every pull request — a
breaking wire-format change must bump the major version and include a migration note in
`CHANGELOG.md`.

### Contributing

Proto changes here affect every consumer independently, so:
- Prefer additive changes (new fields, new RPCs) over renames/removals.
- If a change is unavoidably breaking, call it out in the PR description and expect a major
  version bump.
- `buf lint` and `buf breaking` must pass before merge (enforced in CI).
