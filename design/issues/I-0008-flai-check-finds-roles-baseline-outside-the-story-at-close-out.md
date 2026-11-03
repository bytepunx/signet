---
id: I-0008
title: "flai check finds `roles.baseline` outside the story at close-out"
class: efficiency
status: open
count: 1
first_reported: 2026-11-03T02:48:50Z
last_reported: 2026-11-03T02:48:50Z
updated: 2026-11-03T02:48:50Z
---

# I-0008 flai check finds `roles.baseline` outside the story at close-out

## Description
flai check finds `roles.baseline` outside the story at close-out

## Instances

### 2026-11-03T02:48:50Z
Story: S-0010.
flai check found outside the story:
`design/workflows/roles/analyzer.md`: design/workflows/roles/analyzer.md is missing: flai runs the analyzer from the baseline it carries; run flai upgrade to add the file from the template
`design/workflows/roles/curator.md`: design/workflows/roles/curator.md is missing: flai runs the curator from the baseline it carries; run flai upgrade to add the file from the template
`design/workflows/roles/orchestrator.md`: design/workflows/roles/orchestrator.md is missing: flai runs the orchestrator from the baseline it carries; run flai upgrade to add the file from the template
`design/workflows/roles/planner.md`: design/workflows/roles/planner.md is missing: flai runs the planner from the baseline it carries; run flai upgrade to add the file from the template

## Remediation
