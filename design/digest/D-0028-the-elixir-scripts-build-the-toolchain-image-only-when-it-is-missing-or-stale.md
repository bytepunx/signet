---
id: D-0028
title: The Elixir scripts build the toolchain image only when it is missing or stale, through scripts/ensure-client-elixir-image.sh, so a stalled registry cannot fail a run
kind: cause
sources: [I-0002, S-0021]
topics: [elixir, client-elixir, docker, scripts]
paths: [scripts/ensure-client-elixir-image.sh, scripts/test-client-elixir.sh, scripts/integration-client-elixir.sh, scripts/publish-client-elixir.sh, scripts/test-ensure-client-elixir-image.sh, client-elixir/Dockerfile]
date: 2026-11-06
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0028 The Elixir scripts build the toolchain image only when it is missing or stale, through scripts/ensure-client-elixir-image.sh, so a stalled registry cannot fail a run

I-0002: scripts/test-client-elixir.sh ran `docker build` of the toolchain image on every run, so a stalled registry or credential helper failed the run even when the image was current. S-0021 fixed it.

scripts/ensure-client-elixir-image.sh builds signet-client-elixir:toolchain only when the image is missing, its signet.dockerfile-sha256 label differs from the hash of client-elixir/Dockerfile, or SIGNET_REBUILD=1. The check talks only to the local daemon, so a skipped build never touches the registry. test-client-elixir.sh, integration-client-elixir.sh, and publish-client-elixir.sh all call it; the publish script came from S-0016, accepted while S-0021 ran, and had the same cause. scripts/test-ensure-client-elixir-image.sh, run by scripts/test.sh, checks the skip condition against a fake docker.

I-0003 has the same cause and was left open for the operator to close.

Sources: I-0002, S-0021 (`wip/archive/agents/S-0021.md`, Decisions).
