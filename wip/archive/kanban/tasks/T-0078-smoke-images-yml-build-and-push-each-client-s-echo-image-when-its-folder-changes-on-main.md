---
id: T-0078
type: task
nature: feature
title: "smoke-images.yml: build and push each client's echo image when its folder changes on main"
status: done
parent: S-0019
owner: alex
created: 2026-10-27T22:52:50Z
updated: 2026-11-06T08:48:00Z
transitions:
  - to: ready
    at: 2026-11-06T08:47:55Z
    by: agent-S-0019
  - to: in-progress
    at: 2026-11-06T08:47:55Z
    by: agent-S-0019
  - to: done
    at: 2026-11-06T08:48:00Z
    by: agent-S-0019
stream: S-0019
tags: [ci]
touches: [".github/workflows/smoke-images.yml"]
after: [T-0071]
usage:
  source: log
  seconds: 5
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 2
      output: 423
      cache_read: 63606
      cache_write: 3285
      cost: 0.0237
---
# T-0078 smoke-images.yml: build and push each client's echo image when its folder changes on main

## Work

Add `.github/workflows/smoke-images.yml`. It runs on `push` to `main` with `paths` covering the five clients' folders (`client-go/**`, `client-python/**`, `client-typescript/**`, `client-rust/**`, `client-csharp/**`), the workflow itself, `scripts/build-echo-images.sh`, and, once S-0020 has made the images depend on it, `proto/proto/**`; and on `workflow_dispatch`, which rebuilds all five.

A first job decides, per language, whether that client's folder (or the shared paths above) changed in the push, with a path filter action pinned by SHA or by a script that runs `git diff --name-only` between the push's `before` and `after`. It outputs the list of languages as JSON. A second job runs a matrix over that list and, for each language, logs in to `ghcr.io` with `GITHUB_TOKEN` (`packages: write`, `contents: read`, nothing broader), sets up Buildx, and runs `scripts/build-echo-images.sh <language> --tag latest --push`; it also pushes the image tagged with the commit SHA. A language whose folder did not change builds nothing. A push that changes no client folder starts no matrix job.

The Elixir client is not in the matrix because `client-elixir/examples/echo` does not exist; a comment in the workflow says where to add it and the script already refuses it, so adding it is one matrix value once the example exists.

The workflow calls the script from the previous task, as `tooling.md` asks, and holds no `docker build` inline. It waits for T-0071 because it runs that script.

## Done when

- A push to `main` that changes only `client-python/` builds and pushes `ghcr.io/bytepunx/signet-echo-python:latest` and the commit-SHA tag, and nothing for the other languages, checked in the workflow's run on a pull request branch with `workflow_dispatch` or by reading the generated matrix for a sample diff.
- `workflow_dispatch` builds all five.
- The workflow requests only `contents: read` and `packages: write`, and `actionlint` passes where it is installed.
- No step contains a `docker build` command line; each calls `scripts/build-echo-images.sh`.

## Notes

The old signet-clients `smoke-images.yml` built every image on every push; this one builds only what changed. Reference: the external bytepunx/signet-smoke-test harness pulls `ghcr.io/bytepunx/signet-echo-<language>:latest`. The package visibility on ghcr.io must allow the harness to pull; that is the operator's, named in the docs task.
