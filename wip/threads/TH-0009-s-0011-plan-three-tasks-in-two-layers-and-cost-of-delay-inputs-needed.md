---
id: TH-0009
title: "S-0011 plan: three tasks in two layers, and cost of delay inputs needed"
anchor:
  path: wip/kanban/stories/S-0011-go-client-module-path-for-the-mono-repo-and-release-workflow-on-client-go-v-tags.md
  item: S-0011
status: open
participants: [planner-S-0011]
created: 2026-10-27T22:52:40Z
updated: 2026-10-27T22:52:40Z
---

# TH-0009 S-0011 plan: three tasks in two layers, and cost of delay inputs needed

On wip/kanban/stories/S-0011-go-client-module-path-for-the-mono-repo-and-release-workflow-on-client-go-v-tags.md.

## Entries

### 2026-10-27T22:52:40Z planner-S-0011
Plan for S-0011: touches, a 2h forecast, and three tasks are written; the cost of delay is not.

Question first: cost of delay inputs. Neither S-0011 nor E-0002 has inputs, so `flai cod S-0011` cannot work out a value. Recommended: time_lost_per_cycle 2h, no revenue or penalty. Without this story the Go client stays on the old signet-clients path and client-go has no release workflow. Alternatives: a penalty_per_week figure if importers wait on the new path, or the same inputs as the other E-0002 stories. Please set the inputs on the story or the epic; I write the value on my next run.

Tasks, by layer:

| Layer | Task | Work |
|-------|------|------|
| 1 | T-0056 | Module rename: `go.mod`, `buf.gen.yaml` prefix, imports in 7 `.go` files, regenerate `gen/` |
| 1 | T-0057 | `.github/workflows/release-client-go.yml` on `client-go/v*`: verify, proxy index, GitHub Release |
| 2 | T-0058 | `client-go/README.md` install and publish sections, after T-0056 and T-0057 |

T-0056 and T-0057 share no path, so they run together. T-0058 names the path and the workflow that both settle.

Assumptions:

- The story's `after: S-0007` stands: the workflow calls the shared buf generate and Go client test scripts S-0007 adds. I did not name their files; S-0007 settles them.
- The regenerated stubs keep the BSR pin `buf.build/bytepunx/signet-proto:v0.4.0`; S-0020, which follows this story, moves it to the directory input.
- The generated files embed the Go package path, so T-0056 regenerates them rather than editing them; that needs `buf` and network access to the BSR plugins.
- `CHANGELOG.md` and old issue references (`bytepunx/signet-clients#NN`) stay as they are, per ADR-0001.
- Forecast 2h, raised from flai's 22m for the regenerate step and a workflow that cannot be run before a real tag.
- No task is split, merged, or dropped.
