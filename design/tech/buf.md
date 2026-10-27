---
title: buf
updated: 2026-10-27
status: active
topics: [all, proto]
---

# buf

| | |
|---|---|
| Version | 1.71.0 in `client-elixir/Dockerfile` and as the `bufbuild/buf` image in `scripts/buf-generate.sh` and `scripts/test-proto.sh`, which CI and developers run |
| Used in | `proto/` (lint and breaking checks), every `client-*/buf.gen.yaml` |

## Why

`buf` resolves the `signet-proto` module from the Buf Schema Registry at a pinned tag and
runs code generators, remote on the registry for Go, Python, Rust, TypeScript, and C#,
local for Elixir. Each client pins its own schema version in `buf.gen.yaml`, independent
of the server, so a schema release is adopted deliberately per client.
