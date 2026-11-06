---
id: D-0025
title: "The `roles.definition` finding at close-out is fixed by linking `.claude/agents/{analyzer,curator,orchestrator,planner}.md` to the generated definitions in `.flai-cache`"
kind: cause
sources: [I-0009, S-0032]
topics: [ci, github, release]
paths: [".claude/agents/analyzer.md", ".claude/agents/curator.md", ".claude/agents/orchestrator.md", ".claude/agents/planner.md"]
date: 2026-11-06
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0025 The `roles.definition` finding at close-out is fixed by linking `.claude/agents/{analyzer,curator,orchestrator,planner}.md` to the generated definitions in `.flai-cache`

`flai check` reported `roles.definition` outside the story at close-out of S-0009, S-0010, S-0029, and S-0031 (I-0009, five instances): the four files under `.claude/agents/` were copies whose prompt or description differed from the definition flai generates from `design/workflows/roles/`.

S-0032 fixed it as ADR-0236 and the finding say: run `flai role definition --write`, then replace each of the four files with a link, `ln -sf ../../.flai-cache/agents/<role>.md .claude/agents/<role>.md`. The generated definitions stay in the ignored `.flai-cache`; only the links are committed. A later story that sees the finding again should relink, not edit the copies.

Sources: I-0009, S-0032 (`wip/archive/agents/S-0032.md`, Decisions).
