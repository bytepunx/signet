---
id: D-0023
title: "S-0029: server/v0.11.1 sits on a stray commit and started no release; authorize retag and a push for the path-filter check: I have verified both of these"
kind: answer
sources: [TH-0019]
topics: [release, ci, github, server]
paths: [".github/workflows/publish-server-main.yml", install.sh, ".github/workflows/release-server.yml"]
date: 2026-11-06
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0023 S-0029: server/v0.11.1 sits on a stray commit and started no release; authorize retag and a push for the path-filter check: I have verified both of these

The operator, alex, answered TH-0019 at 2026-11-06T07:32:06Z.

What was asked: after the publish, `server/v0.11.1` (annotated tag dd8c982) pointed at commit 130a92b, which is on no branch and lacks `release-server.yml`, so no release run started and `install.sh` still had nothing to resolve (see I-0011). The agent asked two things: (1) may it delete the stray tag remotely and locally and retag a commit on main that holds `release-server.yml`, or leave 0.11.1 and push `server/v0.11.2`; (2) may it push a docs-only commit touching no `server/` path to main, to see that `publish-server-main.yml` correctly starts no run.

The answer:

```text
I have verified both of these
```

What it settles: the operator says both were verified on their side, so the agent need not retag or push for the path-filter check itself. Criterion 2 still needs a stable `server/v*` release whose workflow succeeded before `install.sh` is run.

Source: TH-0019.
