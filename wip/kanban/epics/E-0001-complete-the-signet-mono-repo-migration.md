---
id: E-0001
type: epic
nature: feature
title: Complete the signet mono-repo migration
status: in-progress
owner: alex
created: 2026-10-24T22:31:44Z
updated: 2026-11-03T18:12:15Z
transitions:
  - to: ready
    at: 2026-10-24T22:33:47Z
    by: claude
  - to: in-progress
    at: 2026-11-03T18:12:15Z
    by: agent-S-0004
tags: []
usage:
  source: sum
  seconds: 3077
  priced_by: harness
  turns:
    - day: 2026-11-03
      ceremony: 7
      test_runs: 7
      hand_edits: 8
      work: 168
  models:
    - model: claude-haiku-5-5
      input: 18
      output: 8859
      cache_read: 298491
      cache_write: 56926
      cost: 0.0145
    - model: claude-opus-5-5
      input: 786
      output: 253644
      cache_read: 51679817
      cache_write: 921887
      cost: 21.762
  strategic:
    - kind: planner
      seconds: 1572
      estimated: true
      models:
        - model: claude-fable-5-1
          input: 1730
          output: 1349
          cache_read: 4725904
          cache_write: 201483
          cost: 6.9792
    - kind: orchestrator
      seconds: 127
      estimated: true
      models:
        - model: claude-opus-5-5
          input: 76
          output: 569
          cache_read: 2136816
          cache_write: 311703
          cost: 1.4005
---
# E-0001 Complete the signet mono-repo migration

## Outcome

The four source repositories (signet, signet-proto, signet-clients, signet-admin) are fully absorbed into this repository. Every sub-project builds, tests, and publishes from here; the proto schema is consumed from the in-repo module; CI, releases, documentation, and design live at the repository root under the system-flow layout; nothing in the tree still points at the old repositories as its home; and the old repositories are archived with a pointer here.

## Stories
- S-0001 Decide proto distribution, Go module paths, and the release model for the mono-repo
- S-0002 Generate every consumer's stubs from the in-repo proto module
- S-0003 Move the Go modules to paths that resolve from this repository
- S-0004 Make the root test, integration, and smoke targets run every sub-project
- S-0005 Rebuild CI as root workflows with one job per sub-project
- S-0006 Release and publish the server, proto, and clients from the mono-repo
- S-0007 Retire stale cross-repo references and sub-project agent files; add the root LICENSE
- S-0008 Fold the server design and security reviews into design/ and retire server/design
- S-0009 Move operator and user documentation from server/ into docs/
- S-0010 Document the client libraries in docs/ and design/
- S-0011 Archive the source repositories with pointers to the mono-repo

## Notes

What the import (commits `db3b802`..`9b60c6c`) left behind, found by survey on 2026-10-24:

- Every `buf.gen.yaml` (server and five clients) still takes its schema from `buf.build/bytepunx/signet-proto:v0.4.0` on the Buf Schema Registry rather than from `proto/`.
- `server/go.mod` is `github.com/bytepunx/signet` while the code sits in `server/`; `client-go/go.mod` is `github.com/bytepunx/signet-clients/go`. Neither path resolves from this repository through the Go module proxy.
- GitHub workflows survive only under `server/.github` and `proto/.github`, where GitHub does not run them. The signet-clients root (`ci.yml`, `proto-bump.yml`, `publish-*.yml`, `release-please.yml`, `smoke-images.yml`, `release-please-config.json`, `.githooks`, `LICENSE`) was not imported at all.
- `server/` and `proto/` each carry a single-package `release-please-config.json` and manifest; the clients' multi-component one was dropped. Tags exist in three namespaces: `v*` (old server), `legacy/<project>/*`, and one `<project>/v*` per project.
- `install.sh` resolves `releases/latest` of `bytepunx/signet`, which stops meaning the server once several components release from one repository.
- `scripts/test.sh`, `integration.sh`, and `smoke.sh` are the template placeholders; `make test` runs nothing.
- `design/system`, `design/tech`, and `docs/*` are template placeholders while the real design (`server/design/draft.md`, two security reviews), operator docs (`server/docs/*`, `server/deploy/docs/*`), and per-client READMEs live inside the sub-projects.
- About 330 mentions of `signet-proto`, `signet-clients`, and `signet-admin` remain across READMEs, `CLAUDE.md` files, package metadata, example Dockerfiles, Helm templates, and test names.
- The repository has no root `LICENSE`; signet-clients was Apache-2.0, signet-proto carries its own, the server never had one.
- `admin/` and `client-erlang/` are documentation-only stubs; nothing to migrate beyond references.
- 2026-10-24T22:33:47Z: moved to ready: follows S-0001, which moved to ready
- 2026-11-03T18:12:15Z: moved to in-progress: follows S-0004, which moved to in-progress
