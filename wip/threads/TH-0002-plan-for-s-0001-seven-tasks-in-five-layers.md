---
id: TH-0002
title: "Plan for S-0001: seven tasks in five layers"
anchor:
  path: wip/kanban/stories/S-0001-decide-proto-distribution-go-module-paths-and-the-release-model-for-the-mono-repo.md
  item: S-0001
status: answered
participants: [planner-S-0001, agent-S-0001]
created: 2026-10-24T22:54:00Z
updated: 2026-11-03T18:26:50Z
---

# TH-0002 Plan for S-0001: seven tasks in five layers

On wip/kanban/stories/S-0001-decide-proto-distribution-go-module-paths-and-the-release-model-for-the-mono-repo.md.

## Entries

### 2026-10-24T22:54:00Z planner-S-0001
S-0001 is enriched and has seven tasks in the backlog.

Enrichment: tags `research, migration, adr, design`; topics `release, proto`; 20 touches (recorded under `### Planning` in the Notes); forecast 6h, delivery 2026-10-25T05:00:00Z, raised from flai's 1h default for want of history; cost of delay value pending the inputs asked for on TH-0001.

Tasks and layers:

| Layer | Tasks | Waits for |
|-------|-------|-----------|
| 1 | T-0001 proto ADR; T-0002 `design/system/overview.md` | nothing |
| 2 | T-0003 Go module paths ADR | T-0001 |
| 3 | T-0004 release model ADR; T-0005 `design/tech/` entries | T-0003; T-0001 |
| 4 | T-0006 licence ADR | T-0004 |
| 5 | T-0007 update the goals of S-0002, S-0003, S-0006 | T-0001, T-0003, T-0004, T-0006 |

The four ADR tasks run one after another because `flai adr new` numbers each file and writes `design/adrs/README.md`, which they all touch; T-0003 also genuinely follows T-0001 (`go_package_prefix` lives in the generators) and T-0004 follows T-0003 (tag namespaces follow the module directories). T-0006 is independent in substance and could move earlier if the agent prefers.

Assumptions:

- `design/adrs/` stays a folder touch on the story: the ADR file names come from `flai adr new` and no task can name them yet. No ready story touches that folder.
- The thirteen `design/tech` files are named now (`go.md`, `buf.md`, `grpc-protobuf.md`, `cockroachdb.md`, `spire.md`, `helm.md`, `sops.md`, `dotnet.md`, `python.md`, `rust.md`, `node-typescript.md`, `sveltekit.md`, `erlang-otp.md`); the agent may rename, widening the touches as it closes the task.
- S-0001 edits no code, generator, `go.mod`, workflow, or `LICENSE`; those are S-0002, S-0003, S-0005, S-0006, and S-0007.
- The release ADR records whether `design/conventions/git.md` stands or needs a project addition; it does not edit the convention.

Proposal: criterion 7 names S-0002, S-0003, and S-0006 only, but the licence ADR bears on S-0007. T-0007 tells the agent to open a thread on S-0007 rather than edit it. If you would rather S-0001 update S-0007's goal too, say so and I will add it to criterion 7 and T-0007.

### 2026-10-24T23:01:42Z planner-S-0001
Correction to the first assumption: S-0008 now carries a `design/adrs` folder touch too (its T-0016 writes ADRs), so whichever of S-0001 and S-0008 is in progress holds the other on that folder. S-0001 is first in the pull order; neither story needs the other's change, so no `after` is set. The story's `### Planning` note records this.

### 2026-11-03T18:26:50Z agent-S-0001
Plan adopted as drafted, with one change: T-0007 no longer waits for T-0006, since the goals of S-0002, S-0003, and S-0006 do not depend on the licence. T-0006 waits on the operator's answer on TH-0009; the S-0007 thread the licence may need moved into T-0006. Recorded in the narrative's Decisions.
