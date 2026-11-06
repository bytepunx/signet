---
id: S-0001
type: story
nature: research
title: Decide proto distribution, Go module paths, and the release model for the mono-repo
status: in-progress
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-11-03T18:46:29Z
transitions:
  - to: ready
    at: 2026-10-24T22:33:47Z
    by: claude
  - to: in-progress
    at: 2026-11-03T18:12:17Z
    by: agent-S-0001
blocked:
  - from: 2026-11-03T18:28:03Z
    until: 2026-11-03T18:45:41Z
    reason: awaiting the operator on TH-0002
  - from: 2026-11-03T18:46:29Z
    reason: awaiting the operator on TH-0002
tags: [research, migration, adr, design]
topics: [release, proto]
touches: [design/adrs, design/adrs/README.md, design/system/overview.md, design/tech/README.md, design/tech/go.md, design/tech/buf.md, design/tech/grpc-protobuf.md, design/tech/cockroachdb.md, design/tech/spire.md, design/tech/helm.md, design/tech/sops.md, design/tech/dotnet.md, design/tech/python.md, design/tech/rust.md, design/tech/node-typescript.md, design/tech/sveltekit.md, design/tech/erlang-otp.md, wip/kanban/stories/S-0002-generate-every-consumer-s-stubs-from-the-in-repo-proto-module.md, wip/kanban/stories/S-0003-move-the-go-modules-to-paths-that-resolve-from-this-repository.md, wip/kanban/stories/S-0006-release-and-publish-the-server-proto-and-clients-from-the-mono-repo.md]
usage:
  source: log
  seconds: 993
  priced_by: harness
  turns:
    - day: 2026-11-03
      ceremony: 3
      hand_edits: 2
      work: 48
  models:
    - model: claude-opus-5-5
      input: 270
      output: 87490
      cache_read: 16219779
      cache_write: 311811
      cost: 7.1035
  strategic:
    - kind: orchestrator
      seconds: 41
      estimated: true
      models:
        - model: claude-opus-5-5
          input: 28
          output: 220
          cache_read: 785675
          cache_write: 98726
          cost: 0.474
forecast:
  duration: 6h
  delivery: 2026-10-25T05:00:00Z
  basis: "No done stories yet, so flai gave planning.default_duration 1h for size 7; raised to 6h for four ADRs each needing a survey of six generators, two go.mod files, eight workflows and three tag namespaces, thirteen tech entries, the overview, and three story edits, with nothing ahead in the pull order."
  by: planner-S-0001
  at: 2026-10-24T22:51:29Z
---
# S-0001 Decide proto distribution, Go module paths, and the release model for the mono-repo

## Goal

Settle, as ADRs, the three decisions every other migration story depends on: how the proto schema reaches its consumers now that it lives in `proto/`; what the Go module paths of `server/` and `client-go/` become; and how components are versioned, tagged, and published from one repository. Record the current design and technology they touch in `design/system` and `design/tech`.

## Acceptance criteria
- [x] An ADR decides how `server/` and each client generate stubs: from `proto/` through a buf workspace or `directory` input, with the Buf Schema Registry module `buf.build/bytepunx/signet-proto` kept, retired, or published as a mirror. It says whether generated code is committed per project (today `server/gen` and `client-go/gen` are committed; C#, Python, and Rust ignore theirs; TypeScript commits `src/gen`).
- [x] An ADR decides the Go module paths for `server/` and `client-go/`, states the `go install` path users get for the `signet` CLI, and says whether a root `go.work` is used.
- [x] An ADR decides the release model: one tag namespace per component (`server/v*`, `proto/v*`, `client-<lang>/v*` already exist alongside legacy `v*` and `legacy/*` tags), what tool computes versions and changelogs (`flai release --pending` per `design/conventions/git.md`, or release-please with a multi-component manifest), how `install.sh` finds the latest server release, and what happens to the legacy tags.
- [ ] An ADR records the repository licence: the root `LICENSE` to add (signet-clients was Apache-2.0, signet-proto has its own file, the server had none) and whether sub-projects keep per-folder copies.
- [x] `design/system/overview.md` describes the real parts of signet (signetd, CLI, proto, the clients, admin, SPIRE, CockroachDB) and how they connect, replacing the template placeholder.
- [x] `design/tech/` has an entry, with version and where used, for every technology the sub-projects depend on: Go, buf, gRPC/protobuf, CockroachDB, SPIRE/go-spiffe, Helm, SOPS, .NET, Python, Rust, Node/TypeScript, SvelteKit, Erlang/OTP.
- [x] The dependent stories (S-0002, S-0003, S-0006) are updated so their goals name the decisions taken.

## Tasks
- T-0001 ADR: generate every consumer's stubs from proto/ and settle the BSR module and the generated-code policy
- T-0002 Write design/system/overview.md: the parts of signet and how they connect
- T-0003 ADR: Go module paths for server/ and client-go/, the signet CLI's go install path, and go.work
- T-0004 ADR: the release model, one tag namespace per component, its versioning tool, install.sh, and the legacy tags
- T-0005 Write design/tech/ entries for every technology the sub-projects depend on
- T-0006 ADR: the repository licence, the root LICENSE, and per-folder copies
- T-0007 Update the goals of S-0002, S-0003, and S-0006 to name the decisions taken

## Notes

Inputs: `server/buf.gen.yaml`, `client-*/buf.gen.yaml`, `server/go.mod`, `client-go/go.mod`, `proto/buf.yaml`, `server/.github/workflows/*`, `proto/.github/workflows/*`, the old signet-clients root workflows on GitHub (`ci.yml`, `proto-bump.yml`, `publish-*.yml`, `release-please.yml`, `smoke-images.yml`, `release-please-config.json`), `install.sh`, and `git tag`. The conventions already say releases are `flai release --pending` with per-component bumps; the ADR should confirm or record the conflict.

### Planning

Planned by planner-S-0001 on 2026-10-24.

Touches. `flai touches suggest S-0001` had nothing to offer: the story declared no touches, and no commit on main has changed `design/adrs`, `design/system/overview.md`, or `design/tech` yet. Every touch below comes from the criteria and the layout:

- `design/adrs/` (folder, kept): the four ADRs are numbered and named by `flai adr new` when the story's agent writes them, so no task can name the files yet. S-0008 carries the same folder touch for its own ADRs, so whichever of the two is in progress holds the other on `design/adrs`; the pull order puts S-0001 first, and neither needs the other's change, so no `after` is set.
- `design/adrs/README.md` (criteria 1 to 4): `flai adr new` adds a row per ADR.
- `design/system/overview.md` (criterion 5, linked from the goal).
- `design/tech/README.md` and one file per technology in criterion 6: `go.md`, `buf.md`, `grpc-protobuf.md`, `cockroachdb.md`, `spire.md`, `helm.md`, `sops.md`, `dotnet.md`, `python.md`, `rust.md`, `node-typescript.md`, `sveltekit.md`, `erlang-otp.md` (layout: `design/tech` is one file per technology, kebab-case).
- The story files of S-0002, S-0003, and S-0006 (criterion 7), edited with `flai edit` in the main checkout.

The story edits no code, no `buf.gen.yaml`, no `go.mod`, and no workflow: it reads them as inputs and leaves their change to S-0002, S-0003, S-0005, and S-0006.

Forecast. `flai forecast` gave 1h, delivery 2026-10-24T23:50:00Z, from `planning.default_duration` for size 7 because no done story has usage yet. Raised to 6h, delivery 2026-10-25T05:00:00Z: four ADRs, each resting on a survey of six generators, two Go modules, eight surviving workflows plus the dropped signet-clients ones on GitHub, and three tag namespaces; thirteen technology entries each with a version looked up in the sub-projects; the overview with a diagram; and three story edits. Nothing is ahead of S-0001 in the pull order, so the delivery follows the duration.

Cost of delay. `flai cod S-0001` could not work out a value: neither S-0001 nor E-0001 has inputs. Asked on TH-0001, recommending `time_lost_per_cycle: 4h` on E-0001; the value is written once the inputs are set.
