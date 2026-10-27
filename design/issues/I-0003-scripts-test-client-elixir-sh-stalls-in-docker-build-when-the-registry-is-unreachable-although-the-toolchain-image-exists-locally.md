---
id: I-0003
title: scripts/test-client-elixir.sh stalls in docker build when the registry is unreachable although the toolchain image exists locally
class: efficiency
status: open
count: 2
cost: 20m
first_reported: 2026-10-27T22:08:02Z
last_reported: 2026-10-27T22:14:42Z
updated: 2026-10-27T22:14:42Z
---

# I-0003 scripts/test-client-elixir.sh stalls in docker build when the registry is unreachable although the toolchain image exists locally

## Description
scripts/test-client-elixir.sh stalls in docker build when the registry is unreachable although the toolchain image exists locally

## Instances

### 2026-10-27T22:08:02Z
Story: S-0002.
During S-0002 (T-0015, T-0016) `docker build -q -t signet-client-elixir:toolchain client-elixir` hung for more than ten minutes resolving the pinned `hexpm/elixir` base image's metadata on Docker Hub, although `signet-client-elixir:toolchain` was already built locally. Every `scripts/test-client-elixir.sh` run, and so `make test`, `flai test`, and the close-out, pays the stall. Workaround used: run the same `docker run --rm -v client-elixir:/app -e MIX_ENV=test signet-client-elixir:toolchain mix ...` steps by hand. Remediation idea: skip the build when the image exists (an env flag or `docker image inspect` check), or build with `--pull=false`.

### 2026-10-27T22:14:42Z
Story: S-0002.
Root cause found at S-0002's close-out: the registry is reachable; what hangs is Docker Desktop's credential helper. `~/.docker/config.json` has `credsStore: desktop`, and every `docker build` or `docker pull` spawns `docker-credential-desktop get`, which never returns in this unattended session (seven stuck copies accumulated), so the client never reaches the daemon. `DOCKER_CONFIG=/tmp/docker-anon` with an empty `{}` config builds the toolchain image in 13s. Remediation: have scripts/test-client-elixir.sh skip the build when the image exists, or document running with an empty DOCKER_CONFIG for unattended agents; the helper's hang is the operator's Docker Desktop setup (keychain prompt with no UI).

## Remediation
