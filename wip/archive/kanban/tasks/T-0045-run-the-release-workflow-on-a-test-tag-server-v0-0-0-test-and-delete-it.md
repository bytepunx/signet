---
id: T-0045
type: task
nature: improvement
title: Run the release workflow on a test tag server/v0.0.0-test and delete it
status: done
parent: S-0009
owner: alex
created: 2026-10-27T22:26:15Z
updated: 2026-11-05T20:45:15Z
transitions:
  - to: ready
    at: 2026-11-05T19:58:58Z
    by: agent-S-0009
  - to: in-progress
    at: 2026-11-05T19:58:58Z
    by: agent-S-0009
  - to: done
    at: 2026-11-05T20:45:06Z
    by: agent-S-0009
stream: S-0009
tags: [server, ci]
touches: [design/issues/I-0006-ci-yml-passes-secrets-buf-token-but-the-secret-set-on-signet-is-bsr-auth-token.md]
after: [T-0043]
usage:
  source: log
  seconds: 2768
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 22
      output: 2486
      cache_read: 736510
      cache_write: 25635
      cost: 0.2011
---
# T-0045 Run the release workflow on a test tag server/v0.0.0-test and delete it

## Work

Prove the release workflow end to end before the first real `server/v*` tag, with no file change.

- Push the story branch, tag its head `server/v0.0.0-test` (annotated, with a short message), push the tag, and watch the `release-server.yml` run to the end with `gh run watch`.
- Check what it produced: four binaries and `sha256sums.txt` on a pre-release GitHub Release named `server/v0.0.0-test`; images tagged `v0.0.0-test` in both registries and no `latest` or `v0.0` change; the chart `signet` version `0.0.0-test` at `oci://ghcr.io/bytepunx/charts`; and the release notes holding the tag annotation.
- Run `SIGNET_VERSION=server/v0.0.0-test sh install.sh` in a container and confirm it downloads and verifies the Linux binary.
- Delete the test tag, the test GitHub Release, the test image tags, and the test chart version, and record in the story's notes what was run and what was deleted.
- This needs the credentials S-0008 sets up (`DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`) and a push to the remote: if either is missing, stop, record the step that failed in the narrative, and ask the operator on a thread instead of working around it.

This task waits for T-0043 because the run exercises all of the story's change, including the documented `install.sh` path.

## Done when

- The `release-server.yml` run on `server/v0.0.0-test` is green and its outputs match the list above.
- The test tag, release, images, and chart version are deleted, and the story's notes record the run's URL.
- `flai check --strict` is clean.

## Notes
