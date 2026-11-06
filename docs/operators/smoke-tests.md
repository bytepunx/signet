---
title: Smoke-test images and the harness
updated: 2026-11-06
status: active
topics: [ci, github]
---

# Smoke-test images and the harness

The external `bytepunx/signet-smoke-test` harness pulls one echo image per client and
proves each client works against a real signet and SPIRE deployment. This repository
builds those images. Release publishing is in [release.md](release.md) and
[design/system/release.md](../../design/system/release.md).

## The images

| Language | Image | Built from |
|----------|-------|------------|
| Go | `ghcr.io/bytepunx/signet-echo-go` | `client-go/examples/echo` |
| Python | `ghcr.io/bytepunx/signet-echo-python` | `client-python/examples/echo` |
| TypeScript | `ghcr.io/bytepunx/signet-echo-typescript` | `client-typescript/examples/echo` |
| Rust | `ghcr.io/bytepunx/signet-echo-rust` | `client-rust/examples/echo` |
| C# | `ghcr.io/bytepunx/signet-echo-csharp` | `client-csharp/examples/echo` |

The images are test-only. They print retrieved secrets to stdout. Never run them in
production.

The Elixir client has no echo example yet. Adding one is one value in the matrix of
`.github/workflows/smoke-images.yml`, plus the language in `scripts/build-echo-images.sh`.

## When they are rebuilt

| Trigger | What is built |
|---------|---------------|
| A push to `main` that changes a client's folder | That client's image |
| A push to `main` that changes `proto/proto/`, `scripts/build-echo-images.sh`, or the workflow | All five |
| Running the `smoke-images` workflow by hand | All five |

Each build is pushed with the tag `latest` and with the commit SHA. A push that changes no
client folder builds nothing.

## What you must set

The workflow needs no secret beyond `GITHUB_TOKEN`. After the first push of each image,
set the package's visibility or access on ghcr.io so the harness can pull it. This is
once per package.

## Running the harness

`make smoke` builds the five images tagged `local`, then runs the harness when
`SIGNET_SMOKE_TEST_DIR` names a checkout of `bytepunx/signet-smoke-test`.

| Needs | From |
|-------|------|
| Docker | Your host |
| A k3d cluster with signet and SPIRE | `bytepunx/kluster` |
| A checkout of the harness | `bytepunx/signet-smoke-test` |

`scripts/smoke.sh` does not know an entry point of the harness that takes an image tag.
It prints where to look and says the harness did not run. Read the output: a zero exit
means the images built.
