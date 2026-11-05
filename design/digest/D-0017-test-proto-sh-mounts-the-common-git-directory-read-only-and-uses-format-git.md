---
id: D-0017
title: "test-proto.sh mounts the common git directory read-only and uses format=git, because in a worktree .git is a pointer file"
kind: decision
sources: [S-0007]
topics: [ci]
paths: [scripts/test-proto.sh]
date: 2026-10-28
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0017 test-proto.sh mounts the common git directory read-only and uses format=git, because in a worktree .git is a pointer file

Decision in S-0007 (wip/archive/agents/S-0007.md): "test-proto.sh mounts the common git directory read-only and uses `format=git`, because in a worktree `.git` is a pointer file."

The buf breaking check against a git input therefore works from a story worktree as well as from the main checkout.
