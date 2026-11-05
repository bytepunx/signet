---
id: T-0040
type: task
nature: feature
title: "Write publish-server-main.yml: push the main image when server/ changes"
status: done
parent: S-0009
owner: alex
created: 2026-10-27T22:25:58Z
updated: 2026-10-28T05:55:45Z
transitions:
  - to: ready
    at: 2026-10-28T05:55:34Z
    by: agent-S-0009
  - to: in-progress
    at: 2026-10-28T05:55:34Z
    by: agent-S-0009
  - to: done
    at: 2026-10-28T05:55:45Z
    by: agent-S-0009
stream: S-0009
tags: [server, ci]
touches: [".github/workflows/publish-server-main.yml"]
usage:
  source: log
  seconds: 11
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 1
      output: 310
      cache_read: 51749
      cache_write: 2711
      cost: 0.0191
---
# T-0040 Write publish-server-main.yml: push the main image when server/ changes

## Work

Create `.github/workflows/publish-server-main.yml`, porting `server/.github/workflows/publish.yml`.

- Trigger on `push` to `main` with `paths: ["server/**", ".github/workflows/publish-server-main.yml"]`, so a push that changes no server file publishes nothing.
- Build with `context: server`, target `signetd`, platforms linux/amd64 and linux/arm64, and push `ghcr.io/bytepunx/signetd:main` and `docker.io/bytepunx/signet:main` with the source and revision labels, logging in to ghcr.io with `GITHUB_TOKEN` and to Docker Hub with `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`.
- Comment that tagged releases use `release-server.yml`.

The old `server/.github/workflows/publish.yml` is retired by S-0006, not here.

## Done when

- The workflow runs on pushes to `main` that change a file under `server/` and on no others.
- It pushes only the `main` tag to both registries.
- The file passes `actionlint` if available, and `flai check --strict` and the markdown lint are clean.

## Notes
