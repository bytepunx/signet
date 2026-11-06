---
id: T-0091
type: task
nature: research
title: "Confirm a stable server/v* release exists and its Release server run succeeded"
status: backlog
parent: S-0029
owner: alex
created: 2026-11-06T06:59:07Z
updated: 2026-11-06T06:59:07Z
transitions: []
stream: S-0029
tags: [server]
touches: [".github/workflows/release-server.yml"]
---
# T-0091 Confirm a stable server/v* release exists and its Release server run succeeded

## Work

Establish the precondition for the `install.sh` check. After `git fetch --tags`, confirm with `gh release list` and `gh run list --workflow release-server.yml` that a stable `server/vX.Y.Z` GitHub release exists, that it was made by a successful Release server run on a commit that contains `release-server.yml`, and that the release carries `signet-<os>-<arch>` assets for linux and darwin on amd64 and arm64 and `sha256sums.txt`.

The tag is the operator's to cut or move (TH-0019): `server/v0.11.1` points at a stray commit that is on no branch and started no run. Do not delete or move a published tag without the operator's answer on that thread. Until it is answered, this task waits.

If the run fails because of the workflow, fix `.github/workflows/release-server.yml` and say in the narrative what failed. If it fails for a secret or token (I-0006), report it on the thread instead.

## Done when

- `gh release list` shows a stable `server/vX.Y.Z` release, and its Release server run is green.
- The release has the four `signet-<os>-<arch>` assets and `sha256sums.txt`.
- The release tag and its run are recorded in the narrative.

## Notes
