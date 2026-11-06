---
id: T-0038
type: task
nature: improvement
title: "Tech record: in-repo consumers generate from the directory, and the final cross-component check"
status: done
parent: S-0020
owner: alex
created: 2026-10-27T22:25:54Z
updated: 2026-11-06T08:41:20Z
transitions:
  - to: ready
    at: 2026-11-06T08:41:18Z
    by: agent-S-0020
  - to: in-progress
    at: 2026-11-06T08:41:18Z
    by: agent-S-0020
  - to: done
    at: 2026-11-06T08:41:20Z
    by: agent-S-0020
stream: S-0020
tags: [proto, server]
touches: [design/tech/buf.md]
after: [T-0028, T-0029, T-0031, T-0032, T-0034, T-0035, T-0036]
usage:
  source: log
  seconds: 2
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 3
      output: 524
      cache_read: 94782
      cache_write: 3672
      cost: 0.0294
---
# T-0038 Tech record: in-repo consumers generate from the directory, and the final cross-component check

## Work

Rewrite the "Why" section of `design/tech/buf.md`, which says each client pins its own schema version from the Buf Schema Registry: the server and the six clients read `../proto/proto` as a directory, the BSR module and its tags are for external consumers only, and the plugins stay pinned in each `buf.gen.yaml`. Fix the "Used in" row too. This task waits for the seven component tasks so the page describes what they did. Then check the whole story: no `buf.gen.yaml` names a `module:` input, `grep -r "buf.build/bytepunx/signet-proto"` finds it only in `proto/`, the Docker build contexts and docs of the clients, and external-consumer text, and `make test` passes.

## Done when

- `design/tech/buf.md` says in-repo consumers generate from the directory and the BSR serves external consumers.
- No `buf.gen.yaml` in the repository has a `module:` input.
- `make test` passes.

## Notes
