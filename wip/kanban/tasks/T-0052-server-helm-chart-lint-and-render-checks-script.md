---
id: T-0052
type: task
nature: feature
title: Server Helm chart lint and render checks script
status: backlog
parent: S-0007
owner: alex
created: 2026-10-27T22:26:58Z
updated: 2026-10-27T22:26:58Z
transitions: []
stream: S-0007
tags: [server, ci]
touches: [scripts/check-server-chart.sh, scripts/test-server.sh]
after: [T-0046]
---
# T-0052 Server Helm chart lint and render checks script

## Work

Add `scripts/check-server-chart.sh`, POSIX `sh` under `set -eu`, that runs the Helm checks of the old `server/.github/workflows/ci.yml` (`helm-lint` job) in a pinned `alpine/helm` image with `server/deploy/helm/signet` mounted, with `signet.trustDomain=example.org` and `signet.existingSecret=signet-secrets`:

1. `helm lint`.
2. `helm template` to validate the values schema.
3. With `admin.clusterAccess` off (the default), the render has `"127.0.0.1:8444"` and no port named `admin`.
4. With `admin.clusterAccess=true` and `admin.tls.acknowledgeInsecure=true`, the render has `"0.0.0.0:8444"` and a port named `admin`.
5. With `admin.clusterAccess=true` and no `admin.tls`, `helm template` fails.

Then edit `scripts/test-server.sh` (T-0046) to call it as a fifth step. It waits for T-0046 because both change `scripts/test-server.sh`.

## Done when

- `scripts/check-server-chart.sh` exits 0 on the current chart and prints each of the five checks.
- Changing the default `SIGNET_ADMIN_ADDR` in a scratch copy of the chart makes check 3 fail.
- `scripts/test-server.sh` runs it last, and `sh -n` is clean on both.

## Notes

Layer 3 of the plan.
