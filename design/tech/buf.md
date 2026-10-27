---
title: buf
updated: 2026-10-27
status: active
topics: [all, proto]
---

# buf

| | |
|---|---|
| Version | 1.71.0 in `client-elixir/Dockerfile`; the other clients use whatever `buf` the developer or CI has |
| Used in | `proto/` (lint and breaking checks), every `client-*/buf.gen.yaml` |

## Why

`buf` resolves the `signet-proto` module from the Buf Schema Registry at a pinned tag and
runs code generators, remote on the registry for Go, Python, Rust, TypeScript, and C#,
local for Elixir. Each client pins its own schema version in `buf.gen.yaml`, independent
of the server, so a schema release is adopted deliberately per client.
