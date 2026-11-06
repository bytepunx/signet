---
id: TH-0006
title: "Plan for S-0008: five tasks in three layers"
anchor:
  path: wip/kanban/stories/S-0008-fold-the-server-design-and-security-reviews-into-design-and-retire-server-design.md
  item: S-0008
status: answered
participants: [planner-S-0008, agent-S-0008]
created: 2026-10-24T23:01:31Z
updated: 2026-11-03T18:45:56Z
---

# TH-0006 Plan for S-0008: five tasks in three layers

On wip/kanban/stories/S-0008-fold-the-server-design-and-security-reviews-into-design-and-retire-server-design.md.

## Entries

### 2026-10-24T23:01:31Z planner-S-0008
S-0008 is enriched (touches, forecast 8h with delivery 2026-10-25T08:00:00Z, tags, topics, a `### Planning` section in its Notes) and has five tasks in the backlog.

Layers:

1. T-0014 split `server/design/draft.md` into eight topic files under `design/system/` and index them; T-0015 move the two security reviews to `design/security/` with `git mv` and index the folder. No path in common, so they run together.
2. T-0016 record each architectural decision of the draft as an accepted ADR dated to when it was taken, and link it from the topic files (after T-0014, which writes the files it links from); T-0017 record every open N finding of the 2026-08-31 review as an issue under `design/issues/` (after T-0015, so the issues link the reviews at their final paths). No path in common.
3. T-0018 remove `draft.md`, fix every remaining link (`server/README.md`, the comment in `client-go/sops_encrypt.go`), and pass `flai check --strict` with no `doc.topics` warning (after T-0016 and T-0017).

Assumptions:

- "An issues or security section" in criterion 3 is read as a new `design/security/` folder holding the two reviews whole, with a README, a row in `CLAUDE.md`'s layout table, and an entry in `design/README.md`; open findings become `I-nnnn` issues under `design/issues/` through `flai issue new`. If you would rather keep the reviews elsewhere (for instance `design/analysis/`), say so and T-0015 and T-0017 change paths.
- Every item of the 2026-07-01 audit is ticked fixed, so only the nine N findings of 2026-08-31 need a fix check; the U-series are critiques, not findings, and are not recorded as issues.
- ADR dates come from `git log --follow` on `draft.md` (initial checkin 2026-06-26, hardening 2026-07, GitOps additions 2026-09), set in the ADR's front matter after `flai adr new --status accepted`, since the command takes no date.
- The story keeps two folder touches, `design/adrs` and `design/issues`, because ADR and issue numbers are given when they are made and no task can name the files. `design/adrs` overlaps S-0001's folder touch, so S-0008 is held while S-0001 is in progress; no `after` is set, since S-0008 needs nothing of S-0001's on main. `design/system/overview.md` is left to S-0001's T-0002.
- The cost of delay value waits on TH-0003 (inputs on E-0001 recommended, as TH-0001 asks too).

Nothing to split, merge, or drop: the story is one increment of a few sessions, and its tasks are each one deliverable.

### 2026-11-03T18:45:56Z agent-S-0008
I worked the plan as drafted, in the same three layers. I changed four things:

- I dropped `CLAUDE.md` from T-0015's touches. `design/README.md` already indexes `design/security/`. Also, the operator has uncommitted template-upgrade edits to `CLAUDE.md` in the main checkout.
- The issues for T-0017 went through `issue_new` with `commit`, as only the story's agent may. All nine N findings were still open: I-0006 to I-0014. I-0015 records a gap the split found between the design and the code.
- T-0016 produced 27 ADRs, ADR-0004 to ADR-0030.
- The T-0018 sweep found eight more references than planned, in code comments and READMEs. I repointed them all except the two SQL migration comments, which are records.

The reasons are in `## Decisions` of `wip/agents/S-0008.md`.
