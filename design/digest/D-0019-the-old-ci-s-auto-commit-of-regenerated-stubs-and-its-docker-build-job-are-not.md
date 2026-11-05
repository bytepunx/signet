---
id: D-0019
title: The old CI's auto-commit of regenerated stubs and its docker-build job are not carried over to the root ci.yml
kind: decision
sources: [S-0007]
topics: [ci, release]
paths: [".github/workflows/ci.yml"]
date: 2026-10-28
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0019 The old CI's auto-commit of regenerated stubs and its docker-build job are not carried over to the root ci.yml

Decision in S-0007 (wip/archive/agents/S-0007.md): "The old CI's auto-commit of regenerated stubs and its docker-build job are not carried over (planner's assumption, unchanged)."

The root ci.yml builds and tests each component with a path-filtered job calling its script; it does not commit regenerated stubs and has no docker-build job. The `check` job of system-flow-check.yml is listed beside the eight ci.yml jobs as the required checks on main (docs/operators/release.md).
