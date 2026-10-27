---
id: TH-0003
title: Plan for S-0008 and the inputs it needs
anchor:
  path: wip/kanban/stories/S-0008-operator-setup-credentials-trusted-publishers-branch-protection-and-archiving-the-old-repositories.md
  item: S-0008
status: open
participants: [planner-S-0008]
created: 2026-10-27T22:26:05Z
updated: 2026-10-27T22:26:05Z
---

# TH-0003 Plan for S-0008 and the inputs it needs

On wip/kanban/stories/S-0008-operator-setup-credentials-trusted-publishers-branch-protection-and-archiving-the-old-repositories.md.

## Entries

### 2026-10-27T22:26:05Z planner-S-0008
Two questions need your answer, my recommendation first.

## Questions

1. Cost of delay inputs. Neither S-0008 nor E-0002 has any. Recommended: `time_lost_per_cycle` 4h. No release workflow can publish until the secrets and trusted publishers exist, so each release cycle waits on this setup and the operator does the publish by hand. Alternatives: a `penalty_per_week`, if you can state one. Set them with `flai edit S-0008 --time-lost-per-cycle 4h`. I will compute the value with `flai cod` on my next run.
2. Branch protection and direct pushes. Acceptance is local (no pull-request acceptance in `system-flow.yaml`), so `flai release --pending` pushes `main` directly. Recommended: require the CI jobs and the system-flow check, do not require pull request reviews, and leave `enforce_admins` off so your direct push still works. Alternative: switch to pull-request acceptance, which is a separate decision and story. T-0025 follows your answer.

## Plan

Four tasks, two layers.

| Layer | Task | Touches |
|-------|------|---------|
| 1 | T-0024 secrets script (`check` and `set`, values from the environment) | `scripts/set-release-secrets.sh` |
| 1 | T-0025 `protect-main.sh` (dry run by default, `--apply`, `--show`) | `scripts/protect-main.sh` |
| 1 | T-0027 `archive-old-repos.sh` (pointer README, then archive; dry run by default) | `scripts/archive-old-repos.sh` |
| 2 | T-0030 runbook checklist in `docs/operators/release.md`, scripts listed in `scripts/README.md` | those two files |

The three scripts share no path, so they run together. T-0030 waits for all three, because it names their flags and the job names.

Story-level change: I added `S-0007` to `after`, beside `S-0005`. `protect-main.sh` must name the job names its accepted `ci.yml` defines, and both stories edit `docs/operators/release.md` and `scripts/README.md`. Remove it if you would rather order them by pull order.

## Assumptions

- S-0005's runbook (on `story/S-0005`) is the base for T-0030. The secret and trusted-publisher names come from it and from `design/system/release.md`. Client stories S-0011 to S-0016 may add more; T-0030 reads them from main.
- Criteria 2 to 4 are your actions, not tasks. The story's agent hands you the checklist and commands (`gh secret set`, `gh api ...protection`, `gh repo archive`) and ticks a criterion only when you report it done. Nothing in this plan creates credentials or archives repositories.
- `ci.yml` must skip a path-filtered job at job level, not at workflow level. A required check that never reports blocks a pull request. S-0007 should keep this in mind; T-0030 records it in the runbook.
- No folder touches are kept.
- The forecast is 2h of agent time against the 1h default, and delivery assumes S-0005 and S-0007 are accepted first.
