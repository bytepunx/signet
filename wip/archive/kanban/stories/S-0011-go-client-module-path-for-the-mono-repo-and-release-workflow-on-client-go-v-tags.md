---
id: S-0011
type: story
nature: feature
title: "Go client: module path for the mono-repo and release workflow on client-go/v* tags"
status: done
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-11-06T08:37:57Z
transitions:
  - to: ready
    at: 2026-10-27T23:01:00Z
    by: alex
  - to: in-progress
    at: 2026-11-06T08:23:38Z
    by: agent-S-0011
  - to: review
    at: 2026-11-06T08:36:33Z
    by: agent-S-0011
  - to: done
    at: 2026-11-06T08:37:57Z
    by: alex
tags: [client-go]
topics: [release, github]
touches: [".github/workflows/release-client-go.yml", client-go/go.mod, client-go/client.go, client-go/restart.go, client-go/restart_test.go, client-go/sops_encrypt.go, client-go/examples/echo/main.go, client-go/examples/getsecret/main.go, client-go/examples/gitops-workload/main.go, client-go/examples/restart-on-change/main.go, client-go/README.md, client-go/buf.gen.yaml, client-go/gen/admin/v1/admin.pb.go, client-go/gen/admin/v1/admin_grpc.pb.go, client-go/gen/signet/v1/secrets.pb.go, client-go/gen/signet/v1/secrets_grpc.pb.go, design/issues/I-0012-concurrent-flai-test-runs-in-one-worktree-corrupt-client-elixir-deps-so-later.md, design/issues/summary.md]
after: [S-0007]
agent:
  harness: claude-code
usage:
  source: log
  seconds: 791
  priced_by: harness
  turns:
    - day: 2026-11-06
      ceremony: 1
      hand_edits: 1
      work: 31
  models:
    - model: claude-fable-5-1
      input: 1452
      output: 27249
      cache_read: 6033740
      cache_write: 170648
      cost: 6.0713
  strategic:
    - kind: orchestrator
      seconds: 24
      estimated: true
      models:
        - model: claude-sonnet-5-5
          input: 16
          output: 108
          cache_read: 357993
          cache_write: 76977
          cost: 0.2275
forecast:
  duration: 2h
  delivery: 2026-11-06T10:58:00Z
  basis: "Its own forecast of 2h; 2nd in the pull order with an in-progress limit of 4, behind S-0020."
  by: flai
  at: 2026-11-06T08:22:49Z
---
# S-0011 Go client: module path for the mono-repo and release workflow on client-go/v* tags

## Goal

The Go client is `github.com/bytepunx/signet/client-go`, resolvable with `go get github.com/bytepunx/signet/client-go@vX.Y.Z` because flai's `client-go/vX.Y.Z` tag is exactly the tag form Go requires for a module in a subdirectory. `release-client-go.yml` runs on that tag: regenerates stubs, vets, tests, asks `proxy.golang.org` to index the version, and creates the GitHub Release. The README's install and publish sections describe the new path and tags.

## Acceptance criteria
- [x] `client-go/go.mod` is `github.com/bytepunx/signet/client-go`, imports and the `go_package_prefix` follow, and `go test ./...` passes
- [x] `release-client-go.yml` triggers on `client-go/v*` only, verifies, indexes the module on the proxy, and creates the GitHub Release with notes from the tag annotation
- [x] `client-go/README.md` names the new module path, the tag form, and the workflow

## Tasks
- T-0056 Rename the Go module to github.com/bytepunx/signet/client-go and regenerate the stubs
- T-0057 Release workflow release-client-go.yml on client-go/v* tags
- T-0058 Rewrite the Go client README install and publish sections for the new path and tags

## Notes

Reference: signet-clients' `publish-go.yml` on GitHub. The module path change is breaking for importers of the old path.

### Planning

Touches, by source:

- Declared: the workflow, `go.mod`, `client.go`, `restart.go`, `sops_encrypt.go`, `README.md`, `buf.gen.yaml`, kept. The folder `client-go/examples` is replaced by the four files that import the old path, found with a grep for `signet-clients/go`.
- Layout: `restart_test.go` imports the old path. The four generated files under `client-go/gen/` carry the package path in their raw descriptors, so regenerating them changes them (the co-change list shows `gen/` files change together).
- Design: `design/system/release.md` gives the workflow's trigger, artifacts, and credential; ADR-0001 gives the tag form and the module path.
- No folder touch is kept.

Figures:

- Forecast 2h, up from flai's 22m (median of three done stories, which were smaller): the stubs need regenerating with buf, and the workflow cannot run until a real tag exists.
- Cost of delay: no inputs on the story or on E-0002, so none is written; the plan's thread asks for them.
- `after: S-0007` stands: the workflow calls the scripts that story adds under `scripts/`. S-0020 follows this story, since it changes the same `buf.gen.yaml`.
