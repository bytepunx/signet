---
id: S-0019
type: story
nature: feature
title: Smoke-test echo images built from the clients' examples
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-10-27T19:47:50Z
transitions: []
tags: []
topics: [ci, github]
touches: [".github/workflows/smoke-images.yml"]
---
# S-0019 Smoke-test echo images built from the clients' examples

## Goal

The external signet-smoke-test harness pulls `ghcr.io/bytepunx/signet-echo-<language>:latest` images built from each client's `examples/echo`. The old signet-clients `smoke-images.yml` built them on every push; the mono-repo rebuilds each image when its client folder changes on `main`, and `scripts/smoke.sh` documents how to run the harness against them.

## Acceptance criteria
- [ ] `smoke-images.yml` builds and pushes the image for each client whose folder changed on `main`, with the Elixir client added once it has an echo example
- [ ] `scripts/smoke.sh` runs or points at the harness and says what it needs

## Tasks

## Notes

Only if the harness is still in use; the operator confirms before this is refined to ready.
