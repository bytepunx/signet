---
id: T-0022
type: task
nature: improvement
title: Delete the dead server/.github folder and the server's release-please files
status: backlog
parent: S-0006
owner: alex
created: 2026-10-27T22:25:13Z
updated: 2026-10-27T22:25:13Z
transitions: []
stream: S-0006
tags: [server]
touches: [server/.github/workflows/ci.yml, server/.github/workflows/publish.yml, server/.github/workflows/release-please.yml, server/.github/workflows/release.yml, server/.github/FUNDING.yml, server/release-please-config.json, server/.release-please-manifest.json]
---
# T-0022 Delete the dead server/.github folder and the server's release-please files

## Work

Delete, with `git rm`, every file under `server/.github/` (the four workflows and `FUNDING.yml`; GitHub reads only the root `.github/`, so none of them runs) and the server's `server/release-please-config.json` and `server/.release-please-manifest.json`. Leave `server/CHANGELOG.md` as the history of the releases release-please made.

Before deleting, check that nothing else reads the deleted files: search `server/` (Makefile, Dockerfile, docs, githooks, deploy) and the root for their names. Fix a reference that would break; leave prose that merely mentions release-please in a README to the story that rewrites that README.

The deleted workflows stay available in git history as the reference for the root workflows of S-0007 and S-0009.

## Done when

- `server/.github/` does not exist.
- `server/release-please-config.json` and `server/.release-please-manifest.json` do not exist.
- No remaining file under `server/` loads or runs a deleted file.
- `flai check --strict` is clean.

## Notes

Layer 1. It shares no path with the other tasks of S-0006, so it runs together with them.
