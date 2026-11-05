---
id: D-0018
title: "The C# test script pins the dotnet sdk:8.0 image, not a patch tag, until the registry can pull the patch tag"
kind: decision
sources: [S-0007, I-0003]
topics: [ci]
paths: [scripts/test-client-csharp.sh]
date: 2026-10-28
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0018 The C# test script pins the dotnet sdk:8.0 image, not a patch tag, until the registry can pull the patch tag

Decision in S-0007 (wip/archive/agents/S-0007.md): "The C# script pins `sdk:8.0`, not a patch tag, because the patch tag could not be pulled; pin it once the registry responds."

This is a temporary pin: a story that touches the script once images pull again should pin the patch tag. Related: D-0015 and I-0003 (docker pull hangs on the development host).
