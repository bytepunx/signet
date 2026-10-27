---
id: T-0086
type: task
nature: feature
title: Operator guide for the smoke-test images and the harness
status: backlog
parent: S-0019
owner: alex
created: 2026-10-27T22:53:05Z
updated: 2026-10-27T22:53:05Z
transitions: []
stream: S-0019
tags: [ci]
touches: [docs/operators/smoke-tests.md, docs/operators/index.md]
after: [T-0078, T-0081]
---
# T-0086 Operator guide for the smoke-test images and the harness

## Work

Add `docs/operators/smoke-tests.md` (front matter `title`, `updated`, `status`) and a row for it in the guide table of `docs/operators/index.md`. It says, in plain tables and short sentences:

- What the images are: `ghcr.io/bytepunx/signet-echo-<language>` for go, python, typescript, rust, and csharp, built from `client-<language>/examples/echo`, test-only, printing secrets to stdout, never to run in production.
- When they are rebuilt: on a push to `main` that changes the client's folder, or by running the `smoke-images` workflow by hand; the tags `latest` and the commit SHA.
- That the Elixir client has no echo example yet, and that adding one is one matrix value in `smoke-images.yml`.
- What the operator must set: the ghcr.io packages' visibility or access for the harness to pull them, set once per package after the first push, and that the workflow needs no secret beyond `GITHUB_TOKEN`.
- How to run the harness: `make smoke`, `SIGNET_SMOKE_TEST_DIR`, what the harness needs (Docker, a k3d cluster from `bytepunx/kluster`), and that `scripts/smoke.sh` says when it did not run the harness.

Link to `release.md` and `../../design/system/release.md` rather than copying them. The task waits for T-0078 and T-0081 because it documents what they built and must not describe a behavior before it exists.

## Done when

- `docs/operators/smoke-tests.md` exists and covers the five points above, with relative links that resolve.
- `docs/operators/index.md` lists it.
- `scripts/lint-md.sh` passes on both files and `flai check --strict` reports nothing new.

## Notes

`docs/operators/release.md` is edited by other stories of E-0002, so this task adds a new file instead of a section there.
