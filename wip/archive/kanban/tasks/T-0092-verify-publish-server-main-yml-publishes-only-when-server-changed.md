---
id: T-0092
type: task
nature: research
title: Verify publish-server-main.yml publishes only when server/ changed
status: done
parent: S-0029
owner: alex
created: 2026-11-06T06:59:10Z
updated: 2026-11-06T07:34:01Z
transitions:
  - to: ready
    at: 2026-11-06T07:33:19Z
    by: agent-S-0029
  - to: in-progress
    at: 2026-11-06T07:33:19Z
    by: agent-S-0029
  - to: done
    at: 2026-11-06T07:34:01Z
    by: agent-S-0029
stream: S-0029
tags: [server]
touches: [".github/workflows/publish-server-main.yml"]
usage:
  source: log
  seconds: 42
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 14
      output: 2211
      cache_read: 921521
      cache_write: 17790
      cost: 0.1854
---
# T-0092 Verify publish-server-main.yml publishes only when server/ changed

## Work

Check criterion 1 against real runs of `.github/workflows/publish-server-main.yml`.

Positive half: with `gh run list --workflow publish-server-main.yml`, confirm that a push to `main` touching `server/` started a run, that the run succeeded, and that the `main` tag now exists on ghcr.io and Docker Hub for the pushed commit (the image revision label equals the commit). The pushes at c341f0a and 970e598 are the candidates.

Negative half: find or make a push to `main` that touches no `server/` path and not the workflow file, and confirm `gh run list` shows no Publish server main run for its commit. Push such a commit only with the operator's go-ahead on TH-0019; the alternative the thread offers is to wait for the push that accepts this story, which touches only `wip/`, and confirm then. If the negative half is left to that push, leave criterion 1 unticked at review and say why in the Notes.

If the filter is wrong, fix `paths` in the workflow and push the fix as a `server/` or workflow change so the next run proves it.

## Done when

- A run for a `server/` push is green and the `main` image carries that commit.
- A push that touched no `server/` path is shown to have started no run, or the Notes say why that half waits for acceptance.
- Criterion 1 is ticked through flai, or left unticked with the reason in the Notes.

## Notes
