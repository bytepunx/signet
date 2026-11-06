---
id: T-0087
type: task
nature: remediation
title: Sweep the remaining separate-repository references and verify the grep criterion
status: done
parent: S-0017
owner: alex
created: 2026-10-27T22:53:09Z
updated: 2026-11-06T08:16:04Z
transitions:
  - to: ready
    at: 2026-11-06T08:15:56Z
    by: agent-S-0017
  - to: in-progress
    at: 2026-11-06T08:15:56Z
    by: agent-S-0017
  - to: done
    at: 2026-11-06T08:16:04Z
    by: agent-S-0017
stream: S-0017
tags: [server, docs]
touches: [server/design/draft.md, server/CHECKLIST.md]
after: [T-0074, T-0079, T-0080]
usage:
  source: log
  seconds: 8
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 5
      output: 712
      cache_read: 216344
      cache_write: 6144
      cost: 0.0533
---
# T-0087 Sweep the remaining separate-repository references and verify the grep criterion

## Work

After the layer-1 tasks, run `grep -rIn "signet-clients\|signet-proto\|signet-admin" --exclude-dir=node_modules --exclude-dir=.git --exclude-dir=wip .` and sort every hit:

- Fix prose that describes a separate repository. `server/design/draft.md` (around line 1100) refers to `bytepunx/signet-clients` as where the GitOps workload clients live: say the `client-*` folders of this repository. Read the whole of `server/design/draft.md` and `server/CHECKLIST.md` for other layout statements, and fix those that are false now.
- Leave, and list in this task's Notes by file group: the BSR module name `buf.build/bytepunx/signet-proto`; `CHANGELOG.md` entries; `design/adrs/0001-...` and `docs/operators/release.md`, which record the old repositories on purpose; the `signet-admin` Kubernetes ServiceAccount and Service names in the server's Helm chart, docs, tests, and CLI (they name a runtime object, not a repository); the `bytepunx/signet-clients#NN` issue references in source comments, which name issues of the archived repository; the Go module path in `client-go` (S-0011); and the `buf.gen.yaml` comments (S-0020).
- If a hit fits none of these and sits outside the story's touches, record it in this task's Notes; do not widen the story silently.

## Done when

- `server/design/draft.md` and `server/CHECKLIST.md` describe no separate repositories.
- This task's Notes list every remaining hit group and the reason it stays.
- `flai check --strict` and the markdown lint pass.

## Notes

Layer 2: it waits for T-0074, T-0079, and T-0080, because it must grep after they have changed their files. It does not wait for T-0070, T-0076, or the contributors guide, which carry no such text.
