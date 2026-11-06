---
id: TH-0014
title: "S-0017 plan: tasks, layers, assumptions, and two questions for the operator"
anchor:
  path: wip/kanban/stories/S-0017-retire-the-old-three-repository-layout-from-agent-instructions-and-readmes.md
  item: S-0017
status: resolved
participants: [planner-S-0017, alex]
created: 2026-10-27T22:53:43Z
updated: 2026-11-06T08:22:48Z
---

# TH-0014 S-0017 plan: tasks, layers, assumptions, and two questions for the operator

On wip/kanban/stories/S-0017-retire-the-old-three-repository-layout-from-agent-instructions-and-readmes.md.

## Entries

### 2026-10-27T22:53:43Z planner-S-0017
Plan for S-0017 (7 tasks, all in the backlog).

Layer 1, run together, no shared path:
- T-0070 overview.md: real parts diagram (server, proto, admin, seven clients).
- T-0074 move server/CLAUDE.md's protocol facts into a new Protocol section of server/README.md; delete server/CLAUDE.md.
- T-0076 server pre-commit hook and `make install-hooks` run from server/ (the hook now lints the whole mono-repo and hooksPath is wrong at the root).
- T-0079 admin/README.md describes the mono-repo; delete admin/CLAUDE.md.
- T-0080 proto/README.md and the seven client READMEs point at proto/, not at the signet-proto repository.

Layer 2:
- T-0085 contributors guide: how each of the ten components is built and tested (after T-0070, T-0074, T-0076, T-0079, whose files it links).
- T-0087 sweep: fix remaining layout prose in server/design/draft.md and server/CHECKLIST.md, then grep and list what stays (after T-0074, T-0079, T-0080).

Assumptions:
- Nothing in server/CLAUDE.md or admin/CLAUDE.md is missing from the root CLAUDE.md except the admin rule "build missing backend capability in server/, not here", which goes to admin/README.md. No project addition to a convention is needed.
- Forecast set to 1h30m (flai said 20m for size 10; seven tasks over ten components justify more).
- No `after` on other stories; S-0006 shares client-elixir/README.md and client-erlang/README.md, and the pull order settles it.

Questions for you:
1. Cost of delay: neither S-0017 nor E-0002 has inputs, so I set no value. My recommendation: revenue_per_week 0, penalty_per_week 0, time_lost_per_cycle 30m (agents read the contradicting server/CLAUDE.md and admin/CLAUDE.md in every session that touches those folders). Give the inputs with `flai edit S-0017 --time-lost-per-cycle 30m` or tell me different figures.
2. Criterion 1 is wider than the goal. A tree-wide grep also finds `bytepunx/signet-clients#NN` issue references in source comments (client-*, server), the `signet-admin` ServiceAccount and Service names in the Helm chart, docs, and tests, the Go module path in client-go (S-0011), the buf.gen.yaml comments (S-0020), ADR-0001, and docs/operators/release.md. I recommend reading the criterion as "no file describes the separate repositories", fixing only prose, and having T-0087 list what stays and why. Say if you want it widened instead (which would take the issue references in many source files and a rename of the ServiceAccount), or reword the criterion yourself.

### 2026-11-06T08:22:48Z alex
Resolved: S-0017 was accepted
