---
id: TH-0010
title: "Plan for S-0014: four tasks in three layers, and the cost of delay inputs"
anchor:
  path: wip/kanban/stories/S-0014-rust-client-release-workflow-on-client-rust-v-tags-publishing-to-crates-io.md
  item: S-0014
status: open
participants: [planner-S-0014]
created: 2026-10-27T22:52:59Z
updated: 2026-10-27T22:52:59Z
---

# TH-0010 Plan for S-0014: four tasks in three layers, and the cost of delay inputs

On wip/kanban/stories/S-0014-rust-client-release-workflow-on-client-rust-v-tags-publishing-to-crates-io.md.

## Entries

### 2026-10-27T22:52:59Z planner-S-0014
Plan for S-0014 (Rust client release workflow). Tasks, in layers:

| Layer | Tasks |
|-------|-------|
| 1 | T-0061 write `.github/workflows/release-client-rust.yml`; T-0062 `client-rust/Cargo.toml` repository, documentation, description |
| 2 | T-0063 `client-rust/README.md` tag form and workflow, after T-0061 |
| 3 | T-0065 run on test tag `client-rust/v0.0.0-test`, then delete it, after T-0062 and T-0063 |

Story `after`: S-0007 only (T-0061 runs `scripts/test-client-rust.sh`). Touches are the three declared files; no folder touch.

Assumptions:

- crates.io cannot delete a version, so a tag with a pre-release suffix (such as the test tag) runs `cargo publish --dry-run` and makes a pre-release GitHub Release. The trigger stays `client-rust/v*` only. Say if you would rather the test tag publish a real pre-release.
- Credential is `CARGO_REGISTRY_TOKEN` (S-0008 sets it), not trusted publishing, as `design/system/release.md` decides.
- `Cargo.toml`'s `version` stays a placeholder that the workflow overwrites.
- Forecast adjusted from flai's 12m to 2h; delivery 2026-10-28T12:30:00Z.

Question, your input (recommended answer first): cost of delay. `flai cod` stopped because neither S-0014 nor E-0002 has inputs. Recommended: set them once on E-0002 so every story under it inherits them, with `flai edit E-0002 --time-lost-per-cycle <duration>` or `--revenue-per-week` / `--penalty-per-week`. I will not guess the figures. Until then S-0014's cost of delay stays unset.
