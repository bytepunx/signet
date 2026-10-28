---
id: D-0015
title: "S-0007: Docker will not pull three images on this host, so the close-out cannot pass here: images are pulled"
kind: answer
sources: [TH-0016]
topics: [ci, github, release, all]
paths: [".github/workflows/ci.yml", scripts/test.sh, scripts/test-server.sh, scripts/test-proto.sh, scripts/test-client-go.sh, scripts/test-client-python.sh, scripts/test-client-typescript.sh, scripts/test-client-rust.sh, scripts/test-client-csharp.sh, scripts/README.md, Makefile, scripts/buf-generate.sh, scripts/check-server-chart.sh, docs/operators/release.md, ".gitignore", design/issues/I-0003-scripts-test-client-elixir-sh-stalls-in-docker-build-when-the-registry-is-unreachable-although-the-toolchain-image-exists-locally.md, design/issues/summary.md, design/issues/I-0004-flai-verify-panics-with-a-nil-pointer-dereference-at-verify-go-244-when-its-test-tier-is-interrupted.md]
date: 2026-10-28
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0015 S-0007: Docker will not pull three images on this host, so the close-out cannot pass here: images are pulled

The operator, alex, answered TH-0016 at 2026-10-28T05:40:04Z.

What was asked:

```text
All 12 tasks are done and committed on story/S-0007, but the close-out stopped at the test tier: `docker pull alpine/helm:3.17.0` hangs on this host (I-0003, bumped to count 3). The same hang blocks `python:3.10-slim` and `rust:1.90-bookworm`. Registry reachability is fine (curl gets 401 from registry-1.docker.io); docker pull itself shows no progress, alone or in parallel.
```

The answer:

```text
images are pulled
```

Source: TH-0016.
