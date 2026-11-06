---
title: buf
updated: 2026-11-06
status: active
topics: [all, proto]
---

# buf

| | |
|---|---|
| Version | 1.71.0 in `client-elixir/Dockerfile` and as the `bufbuild/buf` image in `scripts/buf-generate.sh` and `scripts/test-proto.sh`, which CI and developers run |
| Used in | `proto/` (lint and breaking checks), `server/buf.gen.yaml`, every `client-*/buf.gen.yaml` |

## Why

`buf` runs code generators over the schema in `proto/proto`: remote on the registry for Go,
Python, Rust, TypeScript, and C#, local for Elixir. The server and the six clients read
`../proto/proto` as a directory input in their `buf.gen.yaml`, so a schema change and the
code that uses it land in one pull request. The `signet-proto` module on the Buf Schema
Registry and its tags are for external consumers only. The plugins stay pinned in each
`buf.gen.yaml`, so a plugin release never changes generated output unasked.
