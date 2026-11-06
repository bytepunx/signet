---
id: D-0026
title: "The open blocked interval on archived S-0010 is closed with `flai unblock`, committed on the story branch, because the check on a fresh clone reads the committed `wip/`"
kind: cause
sources: [I-0010, S-0032]
topics: [ci, github, release]
paths: [wip/archive/kanban/stories/S-0010-proto-publish-workflow-bsr-push-from-main-and-label-on-proto-v-tags.md, ".github/workflows/system-flow-check.yml"]
date: 2026-11-06
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0026 The open blocked interval on archived S-0010 is closed with `flai unblock`, committed on the story branch, because the check on a fresh clone reads the committed `wip/`

`flai check` reported `item.blocked-closed` (I-0010, three instances, at the close-outs of S-0009, S-0031, and S-0029): the archived done story S-0010 still had an open blocked interval.

S-0032 ran `flai unblock S-0010` to close it, and committed that change, and its note on S-0013, on the story branch rather than leaving them to the main checkout: the check on a fresh clone, as `.github/workflows/system-flow-check.yml` runs it, reads the committed `wip/`, so an uncommitted change in the main checkout would not clear the finding there.

Sources: I-0010, S-0032 (`wip/archive/agents/S-0032.md`, Decisions).
