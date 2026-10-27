---
id: TH-0015
title: "Plan for S-0016: tasks, layers, assumptions, and cost of delay inputs"
anchor:
  path: wip/kanban/stories/S-0016-elixir-client-release-workflow-on-client-elixir-v-tags-publishing-to-hex-pm.md
  item: S-0016
status: open
participants: [planner-S-0016]
created: 2026-10-27T22:53:46Z
updated: 2026-10-27T22:53:46Z
---

# TH-0015 Plan for S-0016: tasks, layers, assumptions, and cost of delay inputs

On wip/kanban/stories/S-0016-elixir-client-release-workflow-on-client-elixir-v-tags-publishing-to-hex-pm.md.

## Entries

### 2026-10-27T22:53:46Z planner-S-0016
Plan: four tasks in four layers, all in the backlog.

| Layer | Task | Touches |
|-------|------|---------|
| 1 | T-0068 mix.exs reads SIGNET_CLIENT_VERSION (default 0.0.0-dev), source_ref follows the new tag form, CHANGELOG.md leaves the package files | client-elixir/mix.exs |
| 2 | T-0075 scripts/publish-client-elixir.sh runs `mix hex.publish --yes` in the Docker toolchain, with a `--dry-run` mode | scripts/publish-client-elixir.sh, scripts/README.md |
| 3 | T-0082 release-client-elixir.yml: tag trigger, test script, publish script, GitHub Release from the tag annotation | .github/workflows/release-client-elixir.yml |
| 4 | T-0084 delete CHANGELOG.md, README install and Releasing sections, design row in release.md | client-elixir/README.md, client-elixir/CHANGELOG.md, design/system/release.md |

The layers are serial: the script's dry run needs the `mix.exs` change, the workflow calls the script, and the docs name all three.

Assumptions:

- The story's `after: S-0007` stays as declared. Nothing else it needs is unaccepted: `scripts/test-client-elixir.sh` and the Dockerfile are on main.
- The publish commands live in a script, not inline in the workflow, because `design/conventions/tooling.md` asks for it and `scripts/test-client-elixir.sh` passes no environment into the container. This adds `scripts/publish-client-elixir.sh`, `scripts/README.md`, and `design/system/release.md` to the declared touches.
- No test publish is run. Hex releases cannot be deleted after an hour, so the first real run is `client-elixir/v0.1.0`, after S-0008 sets `HEX_API_KEY`. T-0082 checks the workflow with actionlint and the notes commands against a local tag.
- Forecast 2h, adjusted up from flai's 14m (see the story's Notes, Planning).
- Overlaps, not `after` reasons: S-0020 and S-0006 edit other sections of `client-elixir/README.md`; S-0007 and S-0008 add rows to `scripts/README.md`.

Question, the operator's input: S-0016 and E-0002 have no cost of delay inputs, so `flai cod` stopped and the value is unset.

Recommended answer: revenue_per_week 0, penalty_per_week 0, time_lost_per_cycle 1h (each release of the Elixir client is hand-built and hand-published until this lands). Alternatives: time_lost_per_cycle 30m, or penalty_per_week 0 with the same inputs as the other client release stories. Once set, run the planner on S-0016 again to write the value.
