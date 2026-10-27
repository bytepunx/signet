---
id: I-0003
title: scripts/test-client-elixir.sh stalls in docker build when the registry is unreachable although the toolchain image exists locally
class: efficiency
status: open
count: 1
cost: 15m
first_reported: 2026-10-27T22:08:02Z
last_reported: 2026-10-27T22:08:02Z
updated: 2026-10-27T22:08:02Z
---

# I-0003 scripts/test-client-elixir.sh stalls in docker build when the registry is unreachable although the toolchain image exists locally

## Description
scripts/test-client-elixir.sh stalls in docker build when the registry is unreachable although the toolchain image exists locally

## Instances

### 2026-10-27T22:08:02Z
Story: S-0002.
During S-0002 (T-0015, T-0016) `docker build -q -t signet-client-elixir:toolchain client-elixir` hung for more than ten minutes resolving the pinned `hexpm/elixir` base image's metadata on Docker Hub, although `signet-client-elixir:toolchain` was already built locally. Every `scripts/test-client-elixir.sh` run, and so `make test`, `flai test`, and the close-out, pays the stall. Workaround used: run the same `docker run --rm -v client-elixir:/app -e MIX_ENV=test signet-client-elixir:toolchain mix ...` steps by hand. Remediation idea: skip the build when the image exists (an env flag or `docker image inspect` check), or build with `--pull=false`.

## Remediation
