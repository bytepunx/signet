---
id: I-0002
title: scripts/test-client-elixir.sh rebuilds the toolchain image on every run and fails when the registry stalls
class: efficiency
status: open
count: 1
cost: 15m
first_reported: 2026-10-27T21:58:04Z
last_reported: 2026-10-27T21:58:04Z
updated: 2026-10-27T22:31:00Z
---

# I-0002 scripts/test-client-elixir.sh rebuilds the toolchain image on every run and fails when the registry stalls

## Description
scripts/test-client-elixir.sh rebuilds the toolchain image on every run and fails when the registry stalls

## Instances

### 2026-10-27T21:58:04Z
Story: S-0003.
`flai test` runs scripts/test.sh, which runs `docker build` for client-elixir before every test run. Even with signet-client-elixir:toolchain already built, buildkit resolves the base image's metadata from docker.io first; twice during S-0003 (once for T-0019's sub-agent, once for the parent) that resolution hung and the build failed with "DeadlineExceeded: context deadline exceeded" after about 50s, or stalled for 10 minutes, so no test ran although the tests pass against the existing image. Two Docker builds running at once (S-0002 and S-0003) made it worse. Remedy to consider: skip the build when the image exists unless the Dockerfile changed, or pass --pull=false / a build cache.

## Remediation

Story S-0021 remediates this issue, created from it at 2026-10-27T22:31:00Z.
