---
id: T-0001
type: task
nature: research
title: "ADR: generate every consumer's stubs from proto/ and settle the BSR module and the generated-code policy"
status: done
parent: S-0001
owner: alex
created: 2026-10-24T22:52:24Z
updated: 2026-11-03T18:16:48Z
transitions:
  - to: ready
    at: 2026-11-03T18:13:20Z
    by: agent-S-0001
  - to: in-progress
    at: 2026-11-03T18:13:20Z
    by: agent-S-0001
  - to: done
    at: 2026-11-03T18:16:48Z
    by: agent-S-0001
stream: S-0001
tags: [adr, proto, buf]
touches: [design/adrs/README.md, design/adrs/0001-every-consumer-generates-its-stubs-from-proto-through-a-buf-directory-input-the.md]
usage:
  source: log
  seconds: 208
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 23
      output: 7335
      cache_read: 1359832
      cache_write: 26142
      cost: 0.5955
---
# T-0001 ADR: generate every consumer's stubs from proto/ and settle the BSR module and the generated-code policy

## Work

Read `proto/buf.yaml`, `server/buf.gen.yaml`, the five `client-*/buf.gen.yaml`, `proto/.github/workflows/publish.yml` and `release.yml`, and each project's `.gitignore` for its generated folder. Compare the two ways a consumer can read `proto/` from inside the repository: a root `buf.work.yaml` workspace, or a `directory` input in each `buf.gen.yaml`. Decide one, decide whether `buf.build/bytepunx/signet-proto` is retired or kept as a mirror published on `proto/v*` tags, and decide one generated-code policy for all six consumers (committed everywhere, as `server/gen`, `client-go/gen`, and `client-typescript/src/gen` are today, or ignored everywhere, as C#, Python, and Rust do). Record it with `flai adr new` from the story's worktree with `--commit`, which numbers the file and adds its row to `design/adrs/README.md`. Say in Consequences what S-0002 must change.

## Done when

- One ADR under `design/adrs/`, status `accepted`, decides the input mechanism, the fate of the BSR module, and the generated-code policy, with the losing alternatives each given one line.
- The ADR names the six `buf.gen.yaml` files it governs and whether S-0002 commits or ignores generated code per project.
- `design/adrs/README.md` lists it and `flai check --strict` is clean.

## Notes

First of the four ADRs, since the Go module ADR depends on how `go_package_prefix` is managed in `server/buf.gen.yaml` and `client-go/buf.gen.yaml`.
