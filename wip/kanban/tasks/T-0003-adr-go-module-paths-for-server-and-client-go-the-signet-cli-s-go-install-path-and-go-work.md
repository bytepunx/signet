---
id: T-0003
type: task
nature: research
title: "ADR: Go module paths for server/ and client-go/, the signet CLI's go install path, and go.work"
status: done
parent: S-0001
owner: alex
created: 2026-10-24T22:52:45Z
updated: 2026-11-03T18:21:20Z
transitions:
  - to: ready
    at: 2026-11-03T18:17:25Z
    by: agent-S-0001
  - to: in-progress
    at: 2026-11-03T18:17:25Z
    by: agent-S-0001
  - to: done
    at: 2026-11-03T18:21:20Z
    by: agent-S-0001
stream: S-0001
tags: [adr, go]
touches: [design/adrs/README.md, design/adrs/0002-the-go-modules-are-github-com-bytepunx-signet-server-and-github-com-bytepunx.md]
after: [T-0001]
usage:
  source: log
  seconds: 235
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 24
      output: 7748
      cache_read: 1436364
      cache_write: 27613
      cost: 0.6291
---
# T-0003 ADR: Go module paths for server/ and client-go/, the signet CLI's go install path, and go.work

## Work

Read `server/go.mod`, `client-go/go.mod`, `server/buf.gen.yaml` and `client-go/buf.gen.yaml` (`go_package_prefix`), `server/cmd`, `client-go/examples`, and `install.sh`. A Go module in a subdirectory of `github.com/bytepunx/signet` must be `github.com/bytepunx/signet/<dir>` and tag as `<dir>/vX.Y.Z`; the tags `server/v0.11.0` and `client-go/v0.7.0` already exist. Decide the two module paths, state the `go install` path users get for the `signet` CLI (`github.com/bytepunx/signet/server/cmd/signet@server/vX.Y.Z` or an alternative), and decide whether a root `go.work` is used, and if so whether it is committed or ignored. Say how the `go_package_prefix` override follows the new path, consistent with T-0001's generation model. Record it with `flai adr new --commit` from the story's worktree. Say in Consequences what S-0003 must change and that both modules need a module-path note in their `CHANGELOG.md`.

## Done when

- One ADR under `design/adrs/`, status `accepted`, names the module path of `server/` and of `client-go/`, the `go install` path of the CLI, and the `go.work` decision, with the losing alternatives each given one line.
- `design/adrs/README.md` lists it and `flai check --strict` is clean.

## Notes

Waits for T-0001 because the `go_package_prefix` override lives in the generators that ADR governs, and because the two ADR tasks share `design/adrs/README.md`.
