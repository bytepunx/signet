---
id: T-0007
type: task
nature: research
title: Update the goals of S-0002, S-0003, and S-0006 to name the decisions taken
status: done
parent: S-0001
owner: alex
created: 2026-10-24T22:53:42Z
updated: 2026-11-03T18:26:41Z
transitions:
  - to: ready
    at: 2026-11-03T18:25:35Z
    by: agent-S-0001
  - to: in-progress
    at: 2026-11-03T18:25:35Z
    by: agent-S-0001
  - to: done
    at: 2026-11-03T18:26:41Z
    by: agent-S-0001
stream: S-0001
tags: [work-items]
touches: [wip/kanban/stories/S-0002-generate-every-consumer-s-stubs-from-the-in-repo-proto-module.md, wip/kanban/stories/S-0003-move-the-go-modules-to-paths-that-resolve-from-this-repository.md, wip/kanban/stories/S-0006-release-and-publish-the-server-proto-and-clients-from-the-mono-repo.md]
after: [T-0001, T-0003, T-0004]
usage:
  source: log
  seconds: 66
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 20
      output: 6404
      cache_read: 1187210
      cache_write: 22823
      cost: 0.5199
---
# T-0007 Update the goals of S-0002, S-0003, and S-0006 to name the decisions taken

## Work

With the four ADRs accepted, edit the three dependent stories in the main checkout with `flai edit` (or the MCP tool `item_edit`), never by hand: replace "as S-0001 decided" in each goal with the decision and a link to its ADR. S-0002 names the input mechanism (workspace or `directory`), the BSR module's fate, and the generated-code policy per project. S-0003 names both module paths, the CLI's `go install` path, and the `go.work` decision. S-0006 names the tag namespaces, the versioning tool, the `install.sh` lookup, and the legacy tags. Where a criterion says "if the ADR kept ..." or "per the ADR", resolve it to the decided branch. Leave the rest of each story as it was; these are finalized stories, so change only what the decisions settle. Where the licence ADR changes what S-0007 must add, open a thread on S-0007 saying so rather than editing it.

## Done when

- The goals of S-0002, S-0003, and S-0006 each name the decisions and link the ADRs, and no criterion of theirs still hangs on an undecided "if the ADR".
- `flai check --strict` is clean in the main checkout.

## Notes

Last layer: waits for every ADR task (T-0001, T-0003, T-0004, T-0006). `wip/` is written in the main checkout, so this task commits nothing on the story branch; `flai task done` records it.
