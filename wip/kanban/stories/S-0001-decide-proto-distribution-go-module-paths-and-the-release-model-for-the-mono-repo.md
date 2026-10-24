---
id: S-0001
type: story
nature: research
title: Decide proto distribution, Go module paths, and the release model for the mono-repo
status: ready
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-10-24T22:33:47Z
transitions:
  - to: ready
    at: 2026-10-24T22:33:47Z
    by: claude
tags: []
---
# S-0001 Decide proto distribution, Go module paths, and the release model for the mono-repo

## Goal

Settle, as ADRs, the three decisions every other migration story depends on: how the proto schema reaches its consumers now that it lives in `proto/`; what the Go module paths of `server/` and `client-go/` become; and how components are versioned, tagged, and published from one repository. Record the current design and technology they touch in `design/system` and `design/tech`.

## Acceptance criteria
- [ ] An ADR decides how `server/` and each client generate stubs: from `proto/` through a buf workspace or `directory` input, with the Buf Schema Registry module `buf.build/bytepunx/signet-proto` kept, retired, or published as a mirror. It says whether generated code is committed per project (today `server/gen` and `client-go/gen` are committed; C#, Python, and Rust ignore theirs; TypeScript commits `src/gen`).
- [ ] An ADR decides the Go module paths for `server/` and `client-go/`, states the `go install` path users get for the `signet` CLI, and says whether a root `go.work` is used.
- [ ] An ADR decides the release model: one tag namespace per component (`server/v*`, `proto/v*`, `client-<lang>/v*` already exist alongside legacy `v*` and `legacy/*` tags), what tool computes versions and changelogs (`flai release --pending` per `design/conventions/git.md`, or release-please with a multi-component manifest), how `install.sh` finds the latest server release, and what happens to the legacy tags.
- [ ] An ADR records the repository licence: the root `LICENSE` to add (signet-clients was Apache-2.0, signet-proto has its own file, the server had none) and whether sub-projects keep per-folder copies.
- [ ] `design/system/overview.md` describes the real parts of signet (signetd, CLI, proto, the clients, admin, SPIRE, CockroachDB) and how they connect, replacing the template placeholder.
- [ ] `design/tech/` has an entry, with version and where used, for every technology the sub-projects depend on: Go, buf, gRPC/protobuf, CockroachDB, SPIRE/go-spiffe, Helm, SOPS, .NET, Python, Rust, Node/TypeScript, SvelteKit, Erlang/OTP.
- [ ] The dependent stories (S-0002, S-0003, S-0006) are updated so their goals name the decisions taken.

## Tasks

## Notes

Inputs: `server/buf.gen.yaml`, `client-*/buf.gen.yaml`, `server/go.mod`, `client-go/go.mod`, `proto/buf.yaml`, `server/.github/workflows/*`, `proto/.github/workflows/*`, the old signet-clients root workflows on GitHub (`ci.yml`, `proto-bump.yml`, `publish-*.yml`, `release-please.yml`, `smoke-images.yml`, `release-please-config.json`), `install.sh`, and `git tag`. The conventions already say releases are `flai release --pending` with per-component bumps; the ADR should confirm or record the conflict.
