---
id: D-0003
title: "protoc-gen-elixir runs as a `local:` buf plugin in a Docker toolchain image, with generated code gitignored"
kind: decision
sources: [S-0001]
topics: [elixir, client-elixir, proto]
paths: [client-elixir/buf.gen.yaml, client-elixir/Dockerfile, scripts/test-client-elixir.sh, design/tech/protobuf-elixir.md]
date: 2026-10-27
status: absorbed
absorbed_by: "design/tech/protobuf-elixir.md#why"
asked_again: []
cited: []
---

# D-0003 protoc-gen-elixir runs as a `local:` buf plugin in a Docker toolchain image, with generated code gitignored

No BSR plugin exists for Elixir, and the safety convention forbids installing a toolchain on the host. S-0001 therefore named a local `protoc-gen-elixir` escript as a `local:` buf plugin, installed in a Docker toolchain image, and does not commit the generated code. `scripts/test-client-elixir.sh` regenerates it before every build.

Source: S-0001, `wip/archive/agents/S-0001.md`, Decisions.
