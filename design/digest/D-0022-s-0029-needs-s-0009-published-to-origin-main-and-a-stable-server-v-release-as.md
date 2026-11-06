---
id: D-0022
title: "S-0029 needs S-0009 published to origin/main and a stable server/v* release: as of now, there is no longer the CI machinery in git to cut a release, how would I do that?"
kind: answer
sources: [TH-0018]
topics: [release, ci, github, server]
paths: [".github/workflows/publish-server-main.yml", install.sh]
date: 2026-11-05
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0022 S-0029 needs S-0009 published to origin/main and a stable server/v* release: as of now, there is no longer the CI machinery in git to cut a release, how would I do that?

The operator, alex, answered TH-0018 at 2026-11-05T23:44:43Z.

What was asked: S-0029 (verify publish-server-main.yml and install.sh) could not check either criterion. Local main was 115 commits ahead of origin/main, so S-0009's workflows were not on the remote, and no stable `server/vX.Y.Z` release existed for install.sh to resolve. The agent would not push to origin/main itself and asked the operator to publish the accepted work (`git fetch`, then `flai release --pending`) and cut a stable server release.

The answer:

```text
as of now, there is no longer the CI machinery in git to cut a release, how would I do that?
```

What it settles: the operator does not take the publish and the first stable `server/v*` release as their own steps; they are asked how it is done. A story that needs a published workflow or a release on origin must say how to produce it. The follow-up, TH-0019 (D-0023), shows the publish was then done.

Source: TH-0018.
