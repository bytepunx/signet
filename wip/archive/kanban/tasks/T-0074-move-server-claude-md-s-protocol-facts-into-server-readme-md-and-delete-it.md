---
id: T-0074
type: task
nature: remediation
title: Move server/CLAUDE.md's protocol facts into server/README.md and delete it
status: done
parent: S-0017
owner: alex
created: 2026-10-27T22:52:43Z
updated: 2026-11-06T08:15:45Z
transitions:
  - to: ready
    at: 2026-11-06T08:12:48Z
    by: agent-S-0017
  - to: in-progress
    at: 2026-11-06T08:12:48Z
    by: agent-S-0017
  - to: done
    at: 2026-11-06T08:15:45Z
    by: agent-S-0017
stream: S-0017
tags: [server, docs]
touches: [server/CLAUDE.md, server/README.md]
usage:
  source: log
  seconds: 177
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 3
      output: 450
      cache_read: 136718
      cache_write: 3883
      cost: 0.0337
---
# T-0074 Move server/CLAUDE.md's protocol facts into server/README.md and delete it

## Work

`server/CLAUDE.md` predates the root `CLAUDE.md` and contradicts it. Delete it, after moving what is still true.

- Add a `## Protocol` section to `server/README.md`. It says that the gRPC/protobuf schema (`SecretsService`, `AdminService`, `GitOpsService`) lives in `proto/` in this repository and is published to the Buf Schema Registry as `buf.build/bytepunx/signet-proto`; that `server/buf.gen.yaml` generates the server's `gen/` stubs, and `make proto` regenerates them; and that the clients are the `client-*` folders at the repository root, with `admin/` the web UI. Describe `buf.gen.yaml` as it is on this branch: S-0020 may change how it takes its input, so state the input in a sentence that matches the file, or leave the mechanism to `proto/README.md` and link it.
- The Motivation text in `server/CLAUDE.md` repeats the README's opening; drop it. "All design documentation is under the design folder" is in the root `CLAUDE.md`; drop it.
- Delete `server/CLAUDE.md`.

## Done when

- `server/CLAUDE.md` does not exist.
- `server/README.md` has a Protocol section that describes the mono-repo layout and names no separate repository.
- `grep -n "signet-clients\|bytepunx/signet-proto" server/README.md` finds nothing but the BSR module name.
- `flai check --strict` and the markdown lint pass.

## Notes

Layer 1: it shares no path with another task.
