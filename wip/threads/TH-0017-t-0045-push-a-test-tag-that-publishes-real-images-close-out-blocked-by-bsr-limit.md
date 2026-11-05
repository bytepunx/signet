---
id: TH-0017
title: "T-0045: push a test tag that publishes real images? Close-out blocked by BSR limit"
anchor:
  path: wip/kanban/stories/S-0009-server-release-workflow-on-server-v-tags.md
  item: S-0009
status: answered
participants: [agent-S-0009, alex]
created: 2026-10-28T05:57:34Z
updated: 2026-10-29T17:59:11Z
---

# TH-0017 T-0045: push a test tag that publishes real images? Close-out blocked by BSR limit

On wip/kanban/stories/S-0009-server-release-workflow-on-server-v-tags.md.

## Entries

### 2026-10-28T05:57:34Z agent-S-0009
T-0037, T-0039, T-0040, T-0041 and T-0043 are done on story/S-0009. Two things need you.

1. Close-out fails in the test tier on BSR rate limiting (I-0005, bumped). The server's go test -race, lint and make build pass; only the stub-drift step is rate limited. Setting BUF_TOKEN would clear it.

2. T-0045 (criterion 6) pushes story/S-0009 and the tag server/v0.0.0-test, which publishes real images to ghcr.io and Docker Hub. `gh secret list` shows no secrets (DOCKERHUB_USERNAME, DOCKERHUB_TOKEN, BUF_TOKEN look unset). Recommendation: do T-0045 after S-0008's setup is applied.

Questions: (a) May I push the branch and the test tag now, accepting the Docker Hub login will fail without the secrets? (b) Or shall I move S-0009 to review with criterion 6 unticked and noted? Without an answer I will do (b) once close-out passes.

### 2026-10-29T17:59:11Z alex
Docker is already logged into in this local environment and shouldn't need separate variables. The secret for the org that is set on signet is BSR_AUTH_TOKEN, not BUF_TOKEN
