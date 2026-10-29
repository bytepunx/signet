## signet proto

The protobuf schema for the [signet](https://github.com/bytepunx/signet) secrets and
configuration management protocol. It lives in the `proto/` folder of the signet mono-repo.
This folder contains **only** `.proto` sources and buf configuration; no generated code is
committed or published from here.

### What's in here

```
proto/
  buf.yaml                  # buf module: buf.build/bytepunx/signet-proto
  proto/signet/v1/secrets.proto   # SecretsService: secret + config retrieval, watch, restart locks
  proto/admin/v1/admin.proto      # AdminService / GitOpsService: unseal, KEK/SOPS rotation, GitOps sync
```

### Consuming this schema

**In-repo consumers** (`server/` and each `client-*/`) generate their stubs from this
directory, and so always build against the schema at the commit they are on.

**External consumers** pin the schema published to the
[Buf Schema Registry](https://buf.build/bytepunx/signet-proto) by label:

```yaml
# buf.yaml
version: v2
deps:
  - buf.build/bytepunx/signet-proto:v0.4.0
```

Run `buf dep update` to pin the label's commit in your `buf.lock`, then `buf generate` to
produce stubs into your own repo. Each consumer controls its own package layout through
buf's [managed mode](https://buf.build/docs/generate/managed-mode/) in its own
`buf.gen.yaml`; none of the proto files hard-code a `go_package` (or equivalent).

### Publishing and versions

- A push to `main` that changes `proto/` runs `buf push` from this folder
  (`.github/workflows/publish-proto-main.yml`), updating the BSR's default label.
- A release is a tag of the form `proto/vX.Y.Z` (a pre-release is `proto/vX.Y.Z-suffix`).
  Pushing it labels the BSR module `vX.Y.Z`, without the `proto/` prefix, and creates a
  GitHub Release named by the tag with the tag annotation as its notes
  (`.github/workflows/release-proto.yml`).

```bash
git tag -a proto/v0.5.0 -m "notes for the release"
git push origin proto/v0.5.0
```

Both workflows authenticate to the BSR with the repository secret `BUF_TOKEN`.

### Breaking changes

`buf breaking` runs in CI against the last `proto/v*` tag on every pull request. A
breaking wire-format change must bump the major version and include a migration note in
`CHANGELOG.md`.

### Contributing

Proto changes here affect every consumer independently, so:
- Prefer additive changes (new fields, new RPCs) over renames/removals.
- If a change is unavoidably breaking, call it out in the PR description and expect a major
  version bump.
- `buf lint` and `buf breaking` must pass before merge (enforced in CI).
