---
id: TH-0023
title: "T-0065: may I push story/S-0014 and the test tag client-rust/v0.0.0-test to origin?"
anchor:
  path: wip/kanban/stories/S-0014-rust-client-release-workflow-on-client-rust-v-tags-publishing-to-crates-io.md
  item: S-0014
status: open
participants: [agent-S-0014]
created: 2026-11-06T07:53:59Z
updated: 2026-11-06T07:53:59Z
---

# TH-0023 T-0065: may I push story/S-0014 and the test tag client-rust/v0.0.0-test to origin?

On wip/kanban/stories/S-0014-rust-client-release-workflow-on-client-rust-v-tags-publishing-to-crates-io.md.

## Entries

### 2026-11-06T07:53:59Z agent-S-0014
T-0061, T-0062 and T-0063 are done and committed on story/S-0014 (workflow, Cargo.toml, README); acceptance criteria 1-3 are ticked from reading the files (the workflow was YAML-parsed only: actionlint is not installed here; cargo metadata parses Cargo.toml). T-0065 needs a real run, which means pushing the branch and an annotated tag client-rust/v0.0.0-test to origin, which starts Actions and creates a pre-release GitHub Release, then deleting both. The version has a pre-release suffix so the run only does cargo publish --dry-run and never touches crates.io. Recommended: yes, go ahead; I will push, watch the run, delete the Release and tag, and record the run URL. If no, I will leave T-0065 open and move the story to review without that proof; tell me which.
