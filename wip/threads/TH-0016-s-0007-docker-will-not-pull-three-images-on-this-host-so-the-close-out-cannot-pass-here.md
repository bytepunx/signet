---
id: TH-0016
title: "S-0007: Docker will not pull three images on this host, so the close-out cannot pass here"
anchor:
  path: wip/kanban/stories/S-0007-root-ci-workflow-that-builds-and-tests-every-component-on-pull-requests.md
  item: S-0007
status: resolved
participants: [agent-S-0007, alex]
created: 2026-10-28T00:16:50Z
updated: 2026-10-28T05:51:49Z
---

# TH-0016 S-0007: Docker will not pull three images on this host, so the close-out cannot pass here

On wip/kanban/stories/S-0007-root-ci-workflow-that-builds-and-tests-every-component-on-pull-requests.md.

## Entries

### 2026-10-28T00:16:50Z agent-S-0007
All 12 tasks are done and committed on story/S-0007, but the close-out stopped at the test tier: `docker pull alpine/helm:3.17.0` hangs on this host (I-0003, bumped to count 3). The same hang blocks `python:3.10-slim` and `rust:1.90-bookworm`. Registry reachability is fine (curl gets 401 from registry-1.docker.io); docker pull itself shows no progress, alone or in parallel.

Verified by running: buf-generate.sh, test-proto.sh, test-server.sh steps 1-4 (race tests, golangci-lint v1.64.8 via go install, make build, stub drift), the Go, TypeScript and C# scripts, the Python commands (on a local 3.12 image), and ci.yml under actionlint (clean).
Not run: check-server-chart.sh (needs alpine/helm), test-client-rust.sh (needs rust:1.90-bookworm), test-client-python.sh on its pinned image, and so `make test` as a whole. Criteria 1 and 5 stay unticked; 5 needs a GitHub run anyway.

Recommendation: pull `alpine/helm:3.17.0`, `python:3.10-slim`, `rust:1.90-bookworm` on this host (or restart Docker if pulls are stuck), then answer here. I will rerun the close-out and fix whatever those scripts turn up (Rust image version and the helm checks are the likeliest). Alternative: tell me to move S-0007 to review as it stands, with criteria 1 and 5 unticked, and let the first GitHub run exercise the three scripts.

Also: `flai verify` panicked (nil pointer at verify.go:244) when its test tier was interrupted; recorded as I-0004.

### 2026-10-28T05:40:04Z alex
images are pulled

### 2026-10-28T05:51:49Z alex
Resolved: S-0007 was accepted
