---
id: I-0009
title: "flai check finds `roles.definition` outside the story at close-out"
class: efficiency
status: open
count: 1
first_reported: 2026-11-03T02:48:50Z
last_reported: 2026-11-03T02:48:50Z
updated: 2026-11-03T02:48:50Z
---

# I-0009 flai check finds `roles.definition` outside the story at close-out

## Description
flai check finds `roles.definition` outside the story at close-out

## Instances

### 2026-11-03T02:48:50Z
Story: S-0010.
flai check found outside the story:
`.claude/agents/analyzer.md`: .claude/agents/analyzer.md is a copy whose prompt differs from the definition flai generates for the analyzer from the baseline flai carries for design/workflows/roles/analyzer.md; replace it with the link to that definition: run flai role definition --write, then ln -sf ../../.flai-cache/agents/analyzer.md .claude/agents/analyzer.md (ADR-0236)
`.claude/agents/curator.md`: .claude/agents/curator.md is a copy whose prompt differs from the definition flai generates for the curator from the baseline flai carries for design/workflows/roles/curator.md; replace it with the link to that definition: run flai role definition --write, then ln -sf ../../.flai-cache/agents/curator.md .claude/agents/curator.md (ADR-0236)
`.claude/agents/orchestrator.md`: .claude/agents/orchestrator.md is a copy whose description differs from the definition flai generates for the orchestrator from the baseline flai carries for design/workflows/roles/orchestrator.md; replace it with the link to that definition: run flai role definition --write, then ln -sf ../../.flai-cache/agents/orchestrator.md .claude/agents/orchestrator.md (ADR-0236)
`.claude/agents/planner.md`: .claude/agents/planner.md is a copy whose prompt differs from the definition flai generates for the planner from the baseline flai carries for design/workflows/roles/planner.md; replace it with the link to that definition: run flai role definition --write, then ln -sf ../../.flai-cache/agents/planner.md .claude/agents/planner.md (ADR-0236)

## Remediation
